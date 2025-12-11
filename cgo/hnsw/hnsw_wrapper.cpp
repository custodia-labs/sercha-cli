/*
 * hnsw_wrapper.cpp - HNSWlib wrapper implementation for Sercha
 *
 * This wrapper provides a C-compatible interface to HNSWlib for use with CGO.
 * It manages string chunk ID to numeric label mapping internally.
 * Supports configurable storage precision (float32/float16/int8) while
 * keeping float32 for HNSW runtime operations.
 */

#include "hnsw_wrapper.h"
#include <hnswlib/hnswlib.h>
#include <unordered_map>
#include <vector>
#include <string>
#include <fstream>
#include <cstring>
#include <cstdlib>
#include <mutex>
#include <filesystem>
#include <cmath>
#include <algorithm>

// =============================================================================
// Float16 (IEEE 754 half-precision) conversion
// =============================================================================

// Convert float32 to float16 (IEEE 754 half-precision)
static uint16_t float_to_half(float f) {
    uint32_t x;
    std::memcpy(&x, &f, sizeof(float));

    uint32_t sign = (x >> 31) & 0x1;
    int32_t exp = ((x >> 23) & 0xFF) - 127;
    uint32_t mantissa = x & 0x7FFFFF;

    // Handle special cases
    if (exp == 128) {  // inf or NaN
        if (mantissa == 0) {
            return static_cast<uint16_t>((sign << 15) | 0x7C00);  // inf
        }
        return static_cast<uint16_t>((sign << 15) | 0x7E00);  // NaN
    }

    if (exp < -24) {  // too small, flush to zero
        return static_cast<uint16_t>(sign << 15);
    }

    if (exp < -14) {  // denormalized
        mantissa |= 0x800000;  // add implicit bit
        int shift = -14 - exp;
        mantissa >>= shift;
        return static_cast<uint16_t>((sign << 15) | (mantissa >> 13));
    }

    if (exp > 15) {  // overflow to inf
        return static_cast<uint16_t>((sign << 15) | 0x7C00);
    }

    // Normalized
    uint16_t h_exp = static_cast<uint16_t>(exp + 15);
    uint16_t h_mantissa = static_cast<uint16_t>(mantissa >> 13);
    return static_cast<uint16_t>((sign << 15) | (h_exp << 10) | h_mantissa);
}

// Convert float16 to float32
static float half_to_float(uint16_t h) {
    uint32_t sign = (h >> 15) & 0x1;
    uint32_t exp = (h >> 10) & 0x1F;
    uint32_t mantissa = h & 0x3FF;

    uint32_t result;

    if (exp == 0) {  // zero or denormalized
        if (mantissa == 0) {
            result = sign << 31;
        } else {
            // Denormalized: normalize it
            exp = 1;
            while ((mantissa & 0x400) == 0) {
                mantissa <<= 1;
                exp--;
            }
            mantissa &= 0x3FF;
            result = (sign << 31) | ((exp + 127 - 15) << 23) | (mantissa << 13);
        }
    } else if (exp == 31) {  // inf or NaN
        result = (sign << 31) | 0x7F800000 | (mantissa << 13);
    } else {  // normalized
        result = (sign << 31) | ((exp + 127 - 15) << 23) | (mantissa << 13);
    }

    float f;
    std::memcpy(&f, &result, sizeof(float));
    return f;
}

// =============================================================================
// Int8 symmetric quantization (per-vector scale)
// =============================================================================

// Quantize a float32 vector to int8 with per-vector symmetric scaling
// Returns the scale factor used
static float quantize_vector_int8(const float* in, int8_t* out, int dim) {
    // Find max absolute value
    float max_abs = 0.0f;
    for (int i = 0; i < dim; i++) {
        max_abs = std::max(max_abs, std::abs(in[i]));
    }

    // Compute scale (map max_abs to 127)
    float scale = (max_abs > 0) ? max_abs / 127.0f : 1.0f;
    float inv_scale = (max_abs > 0) ? 127.0f / max_abs : 0.0f;

    // Quantize
    for (int i = 0; i < dim; i++) {
        float val = in[i] * inv_scale;
        val = std::max(-127.0f, std::min(127.0f, val));
        out[i] = static_cast<int8_t>(std::round(val));
    }

    return scale;
}

// Dequantize an int8 vector back to float32
static void dequantize_vector_int8(const int8_t* in, float* out, int dim, float scale) {
    for (int i = 0; i < dim; i++) {
        out[i] = static_cast<float>(in[i]) * scale;
    }
}

// =============================================================================
// Internal structure holding the HNSW index and ID mappings
// =============================================================================

struct HnswIndex {
    hnswlib::InnerProductSpace* space;
    hnswlib::HierarchicalNSW<float>* hnsw;
    std::unordered_map<std::string, hnswlib::labeltype> id_to_label;
    std::vector<std::string> label_to_id;
    std::string path;
    int dimension;
    size_t max_elements;
    hnswlib::labeltype next_label;
    std::mutex mutex;
    bool modified;
    HnswPrecision precision;  // Storage precision
};

// Helper: normalize vector for cosine similarity via inner product
static void normalize_vector(float* vec, int dim) {
    float norm = 0.0f;
    for (int i = 0; i < dim; i++) {
        norm += vec[i] * vec[i];
    }
    norm = std::sqrt(norm);
    if (norm > 0) {
        for (int i = 0; i < dim; i++) {
            vec[i] /= norm;
        }
    }
}

// Helper: save ID mappings to file (includes precision metadata)
static bool save_id_mappings(HnswIndex* idx) {
    std::string mapping_path = idx->path + "/id_mapping.bin";
    std::ofstream out(mapping_path, std::ios::binary);
    if (!out.is_open()) {
        return false;
    }

    // Write precision (new field)
    int32_t prec = static_cast<int32_t>(idx->precision);
    out.write(reinterpret_cast<const char*>(&prec), sizeof(prec));

    // Write number of mappings
    size_t count = idx->label_to_id.size();
    out.write(reinterpret_cast<const char*>(&count), sizeof(count));

    // Write next_label
    out.write(reinterpret_cast<const char*>(&idx->next_label), sizeof(idx->next_label));

    // Write each mapping: label, string length, string data
    for (size_t i = 0; i < count; i++) {
        hnswlib::labeltype label = static_cast<hnswlib::labeltype>(i);
        const std::string& id = idx->label_to_id[i];
        size_t len = id.size();
        out.write(reinterpret_cast<const char*>(&label), sizeof(label));
        out.write(reinterpret_cast<const char*>(&len), sizeof(len));
        out.write(id.data(), len);
    }

    return out.good();
}

// Helper: save vectors in compressed format (float16 or int8)
static bool save_compressed_vectors(HnswIndex* idx) {
    if (idx->precision == HNSW_PRECISION_FLOAT32) {
        return true;  // No separate file needed for float32
    }

    std::string vec_path;
    if (idx->precision == HNSW_PRECISION_FLOAT16) {
        vec_path = idx->path + "/vectors.f16";
    } else {
        vec_path = idx->path + "/vectors.i8";
    }

    std::ofstream out(vec_path, std::ios::binary);
    if (!out.is_open()) {
        return false;
    }

    // Write header
    uint32_t num_vectors = static_cast<uint32_t>(idx->label_to_id.size());
    uint32_t dimensions = static_cast<uint32_t>(idx->dimension);
    out.write(reinterpret_cast<const char*>(&num_vectors), sizeof(num_vectors));
    out.write(reinterpret_cast<const char*>(&dimensions), sizeof(dimensions));

    // Write each vector
    for (size_t label = 0; label < idx->label_to_id.size(); label++) {
        if (idx->label_to_id[label].empty()) {
            // Deleted entry - write zeros
            if (idx->precision == HNSW_PRECISION_FLOAT16) {
                std::vector<uint16_t> zeros(idx->dimension, 0);
                out.write(reinterpret_cast<const char*>(zeros.data()),
                         idx->dimension * sizeof(uint16_t));
            } else {
                float zero_scale = 0.0f;
                out.write(reinterpret_cast<const char*>(&zero_scale), sizeof(zero_scale));
                std::vector<int8_t> zeros(idx->dimension, 0);
                out.write(reinterpret_cast<const char*>(zeros.data()), idx->dimension);
            }
            continue;
        }

        // Get float32 vector from HNSW
        std::vector<float> vec(idx->dimension);
        try {
            std::vector<float> data_vec = idx->hnsw->getDataByLabel<float>(label);
            std::memcpy(vec.data(), data_vec.data(), idx->dimension * sizeof(float));
        } catch (...) {
            // Label not found, write zeros
            if (idx->precision == HNSW_PRECISION_FLOAT16) {
                std::vector<uint16_t> zeros(idx->dimension, 0);
                out.write(reinterpret_cast<const char*>(zeros.data()),
                         idx->dimension * sizeof(uint16_t));
            } else {
                float zero_scale = 0.0f;
                out.write(reinterpret_cast<const char*>(&zero_scale), sizeof(zero_scale));
                std::vector<int8_t> zeros(idx->dimension, 0);
                out.write(reinterpret_cast<const char*>(zeros.data()), idx->dimension);
            }
            continue;
        }

        // Compress and write
        if (idx->precision == HNSW_PRECISION_FLOAT16) {
            std::vector<uint16_t> compressed(idx->dimension);
            for (int i = 0; i < idx->dimension; i++) {
                compressed[i] = float_to_half(vec[i]);
            }
            out.write(reinterpret_cast<const char*>(compressed.data()),
                     idx->dimension * sizeof(uint16_t));
        } else {  // INT8
            std::vector<int8_t> compressed(idx->dimension);
            float scale = quantize_vector_int8(vec.data(), compressed.data(), idx->dimension);
            out.write(reinterpret_cast<const char*>(&scale), sizeof(scale));
            out.write(reinterpret_cast<const char*>(compressed.data()), idx->dimension);
        }
    }

    return out.good();
}

// Helper: load compressed vectors and reconstruct HNSW index
static bool load_compressed_vectors(HnswIndex* idx) {
    std::string vec_path;
    if (idx->precision == HNSW_PRECISION_FLOAT16) {
        vec_path = idx->path + "/vectors.f16";
    } else if (idx->precision == HNSW_PRECISION_INT8) {
        vec_path = idx->path + "/vectors.i8";
    } else {
        return true;  // Float32 doesn't use separate vector file
    }

    if (!std::filesystem::exists(vec_path)) {
        return false;
    }

    std::ifstream in(vec_path, std::ios::binary);
    if (!in.is_open()) {
        return false;
    }

    // Read header
    uint32_t num_vectors, dimensions;
    in.read(reinterpret_cast<char*>(&num_vectors), sizeof(num_vectors));
    in.read(reinterpret_cast<char*>(&dimensions), sizeof(dimensions));

    if (static_cast<int>(dimensions) != idx->dimension) {
        return false;
    }

    // Read and add each vector
    for (uint32_t label = 0; label < num_vectors; label++) {
        std::vector<float> vec(idx->dimension);

        if (idx->precision == HNSW_PRECISION_FLOAT16) {
            std::vector<uint16_t> compressed(idx->dimension);
            in.read(reinterpret_cast<char*>(compressed.data()),
                   idx->dimension * sizeof(uint16_t));
            for (int i = 0; i < idx->dimension; i++) {
                vec[i] = half_to_float(compressed[i]);
            }
        } else {  // INT8
            float scale;
            in.read(reinterpret_cast<char*>(&scale), sizeof(scale));
            std::vector<int8_t> compressed(idx->dimension);
            in.read(reinterpret_cast<char*>(compressed.data()), idx->dimension);
            dequantize_vector_int8(compressed.data(), vec.data(), idx->dimension, scale);
        }

        // Only add if this label has a valid ID
        if (label < idx->label_to_id.size() && !idx->label_to_id[label].empty()) {
            try {
                idx->hnsw->addPoint(vec.data(), label);
            } catch (...) {
                // Ignore errors for individual vectors
            }
        }
    }

    return in.good();
}

// Helper: load ID mappings from file (includes precision metadata)
static bool load_id_mappings(HnswIndex* idx) {
    std::string mapping_path = idx->path + "/id_mapping.bin";
    std::ifstream in(mapping_path, std::ios::binary);
    if (!in.is_open()) {
        return false;
    }

    // Read precision (new field)
    int32_t prec;
    in.read(reinterpret_cast<char*>(&prec), sizeof(prec));
    idx->precision = static_cast<HnswPrecision>(prec);

    // Read number of mappings
    size_t count;
    in.read(reinterpret_cast<char*>(&count), sizeof(count));

    // Read next_label
    in.read(reinterpret_cast<char*>(&idx->next_label), sizeof(idx->next_label));

    // Read each mapping
    idx->label_to_id.resize(count);
    idx->id_to_label.clear();
    for (size_t i = 0; i < count; i++) {
        hnswlib::labeltype label;
        size_t len;
        in.read(reinterpret_cast<char*>(&label), sizeof(label));
        in.read(reinterpret_cast<char*>(&len), sizeof(len));

        std::string id(len, '\0');
        in.read(&id[0], len);

        if (label < idx->label_to_id.size()) {
            idx->label_to_id[label] = id;
            if (!id.empty()) {
                idx->id_to_label[id] = label;
            }
        }
    }

    return in.good();
}

extern "C" {

HnswIndex* hnsw_create(const char* path, int dimension, int max_elements, HnswPrecision precision) {
    if (path == nullptr || dimension <= 0 || max_elements <= 0) {
        return nullptr;
    }

    try {
        // Create directory if it doesn't exist
        std::filesystem::create_directories(path);

        HnswIndex* idx = new HnswIndex();
        idx->path = path;
        idx->dimension = dimension;
        idx->max_elements = static_cast<size_t>(max_elements);
        idx->next_label = 0;
        idx->modified = false;
        idx->precision = precision;

        // Use inner product space (cosine similarity on normalized vectors)
        idx->space = new hnswlib::InnerProductSpace(dimension);

        // HNSW parameters: M=16, ef_construction=200
        idx->hnsw = new hnswlib::HierarchicalNSW<float>(
            idx->space,
            idx->max_elements,
            16,   // M - number of connections per element
            200   // ef_construction - controls index quality
        );

        // Set ef for search (controls recall vs speed)
        idx->hnsw->setEf(50);

        return idx;
    } catch (...) {
        return nullptr;
    }
}

HnswIndex* hnsw_open(const char* path, int dimension) {
    if (path == nullptr || dimension <= 0) {
        return nullptr;
    }

    std::string index_path = std::string(path) + "/index.bin";
    std::string mapping_path = std::string(path) + "/id_mapping.bin";

    // Check if mapping file exists (needed to determine precision)
    if (!std::filesystem::exists(mapping_path)) {
        return nullptr;
    }

    try {
        HnswIndex* idx = new HnswIndex();
        idx->path = path;
        idx->dimension = dimension;
        idx->next_label = 0;
        idx->modified = false;
        idx->precision = HNSW_PRECISION_FLOAT32;  // Default, will be overwritten

        // Use inner product space
        idx->space = new hnswlib::InnerProductSpace(dimension);

        // Load ID mappings first (this sets idx->precision from stored value)
        if (!load_id_mappings(idx)) {
            delete idx->space;
            delete idx;
            return nullptr;
        }

        // For float32, load the full index directly
        if (idx->precision == HNSW_PRECISION_FLOAT32) {
            if (!std::filesystem::exists(index_path)) {
                delete idx->space;
                delete idx;
                return nullptr;
            }
            idx->hnsw = new hnswlib::HierarchicalNSW<float>(idx->space, index_path);
            idx->max_elements = idx->hnsw->max_elements_;
        } else {
            // For compressed storage, create empty HNSW and load vectors
            size_t max_elements = idx->label_to_id.size();
            if (max_elements == 0) max_elements = 100000;  // Default
            idx->max_elements = max_elements;

            idx->hnsw = new hnswlib::HierarchicalNSW<float>(
                idx->space, idx->max_elements, 16, 200);

            // Load compressed vectors and add to HNSW
            if (!load_compressed_vectors(idx)) {
                delete idx->hnsw;
                delete idx->space;
                delete idx;
                return nullptr;
            }
        }

        // Set ef for search
        idx->hnsw->setEf(50);

        return idx;
    } catch (...) {
        return nullptr;
    }
}

int hnsw_add(HnswIndex* index, const char* chunk_id, const float* vector, int dimension) {
    if (index == nullptr || chunk_id == nullptr || vector == nullptr) {
        return -1;
    }

    if (dimension != index->dimension) {
        return -1;
    }

    std::lock_guard<std::mutex> lock(index->mutex);

    try {
        std::string id(chunk_id);

        // Check if this ID already exists (update case)
        auto it = index->id_to_label.find(id);
        hnswlib::labeltype label;

        if (it != index->id_to_label.end()) {
            // Update existing: mark old as deleted, add new
            label = it->second;
            index->hnsw->markDelete(label);
        }

        // Assign new label
        label = index->next_label++;

        // Normalize vector for cosine similarity
        std::vector<float> normalized(vector, vector + dimension);
        normalize_vector(normalized.data(), dimension);

        // Check if we need to resize
        if (label >= index->max_elements) {
            index->hnsw->resizeIndex(index->max_elements * 2);
            index->max_elements *= 2;
        }

        // Add to index
        index->hnsw->addPoint(normalized.data(), label);

        // Update mappings
        if (label >= index->label_to_id.size()) {
            index->label_to_id.resize(label + 1);
        }
        index->label_to_id[label] = id;
        index->id_to_label[id] = label;
        index->modified = true;

        return 0;
    } catch (...) {
        return -1;
    }
}

int hnsw_delete(HnswIndex* index, const char* chunk_id) {
    if (index == nullptr || chunk_id == nullptr) {
        return -1;
    }

    std::lock_guard<std::mutex> lock(index->mutex);

    try {
        std::string id(chunk_id);
        auto it = index->id_to_label.find(id);

        if (it == index->id_to_label.end()) {
            // ID not found - not an error, just no-op
            return 0;
        }

        hnswlib::labeltype label = it->second;

        // Mark as deleted in HNSW
        index->hnsw->markDelete(label);

        // Remove from mappings
        index->id_to_label.erase(it);
        if (label < index->label_to_id.size()) {
            index->label_to_id[label] = "";  // Clear but keep slot
        }
        index->modified = true;

        return 0;
    } catch (...) {
        return -1;
    }
}

int hnsw_search(HnswIndex* index, const float* query, int dimension, int k,
                HnswSearchResult** results) {
    if (index == nullptr || query == nullptr || results == nullptr || k <= 0) {
        return -1;
    }

    if (dimension != index->dimension) {
        return -1;
    }

    std::lock_guard<std::mutex> lock(index->mutex);

    try {
        // Normalize query vector
        std::vector<float> normalized(query, query + dimension);
        normalize_vector(normalized.data(), dimension);

        // Search
        auto result = index->hnsw->searchKnn(normalized.data(), k);

        // Count valid results (non-deleted entries)
        std::vector<std::pair<float, hnswlib::labeltype>> valid_results;
        while (!result.empty()) {
            auto item = result.top();
            result.pop();

            hnswlib::labeltype label = item.second;
            if (label < index->label_to_id.size() && !index->label_to_id[label].empty()) {
                valid_results.push_back(item);
            }
        }

        if (valid_results.empty()) {
            *results = nullptr;
            return 0;
        }

        // Allocate results array
        int count = static_cast<int>(valid_results.size());
        *results = static_cast<HnswSearchResult*>(malloc(sizeof(HnswSearchResult) * count));
        if (*results == nullptr) {
            return -1;
        }

        // Fill results (reverse order since priority_queue gives largest first)
        for (int i = 0; i < count; i++) {
            int idx = count - 1 - i;  // Reverse to get best first
            auto& item = valid_results[idx];

            hnswlib::labeltype label = item.second;
            const std::string& chunk_id = index->label_to_id[label];

            (*results)[i].chunk_id = strdup(chunk_id.c_str());
            // Inner product similarity is already in [0,1] for normalized vectors
            (*results)[i].similarity = 1.0f - item.first;  // Convert distance to similarity
        }

        return count;
    } catch (...) {
        return -1;
    }
}

void hnsw_free_results(HnswSearchResult* results, int count) {
    if (results != nullptr) {
        for (int i = 0; i < count; i++) {
            free(results[i].chunk_id);
        }
        free(results);
    }
}

void hnsw_close(HnswIndex* index) {
    if (index == nullptr) {
        return;
    }

    std::lock_guard<std::mutex> lock(index->mutex);

    try {
        // Save index and mappings if modified
        if (index->modified) {
            // Always save ID mappings (includes precision metadata)
            save_id_mappings(index);

            if (index->precision == HNSW_PRECISION_FLOAT32) {
                // For float32, save the full HNSW index
                std::string index_path = index->path + "/index.bin";
                index->hnsw->saveIndex(index_path);
            } else {
                // For float16/int8, save compressed vectors
                save_compressed_vectors(index);
            }
        }

        delete index->hnsw;
        delete index->space;
        delete index;
    } catch (...) {
        // Best effort cleanup
        delete index;
    }
}

} // extern "C"
