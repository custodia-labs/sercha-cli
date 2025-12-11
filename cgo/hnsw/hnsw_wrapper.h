#ifndef SERCHA_HNSW_WRAPPER_H
#define SERCHA_HNSW_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stddef.h>
#include <stdint.h>

// Opaque handle for the HNSW index.
typedef struct HnswIndex HnswIndex;

// SearchResult holds a single search result.
typedef struct {
    char* chunk_id;
    float similarity;
} HnswSearchResult;

// Storage precision for vectors (runtime always uses float32).
typedef enum {
    HNSW_PRECISION_FLOAT32 = 0,  // 4 bytes per dimension (no compression)
    HNSW_PRECISION_FLOAT16 = 1,  // 2 bytes per dimension (50% savings)
    HNSW_PRECISION_INT8 = 2      // 1 byte per dimension (75% savings)
} HnswPrecision;

// Create a new HNSW index with specified storage precision.
// Returns NULL on error.
HnswIndex* hnsw_create(const char* path, int dimension, int max_elements, HnswPrecision precision);

// Open an existing HNSW index.
// Returns NULL on error.
HnswIndex* hnsw_open(const char* path, int dimension);

// Add a vector to the index.
// Returns 0 on success, -1 on error.
int hnsw_add(HnswIndex* index, const char* chunk_id, const float* vector, int dimension);

// Delete a vector from the index.
// Returns 0 on success, -1 on error.
int hnsw_delete(HnswIndex* index, const char* chunk_id);

// Search for the k nearest neighbors.
// Returns the number of results, or -1 on error.
int hnsw_search(HnswIndex* index, const float* query, int dimension, int k,
                HnswSearchResult** results);

// Free search results.
void hnsw_free_results(HnswSearchResult* results, int count);

// Close and free the index.
void hnsw_close(HnswIndex* index);

#ifdef __cplusplus
}
#endif

#endif // SERCHA_HNSW_WRAPPER_H
