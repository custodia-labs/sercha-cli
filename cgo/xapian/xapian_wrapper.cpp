/*
 * xapian_wrapper.cpp - C-compatible wrapper implementation for Xapian C++ API
 *
 * This implementation wraps Xapian's C++ API in C-compatible functions for use with CGO.
 * Error handling uses a thread-local error string accessible via xapian_get_error().
 */

#include "xapian_wrapper.h"
#include <xapian.h>
#include <string>
#include <cstring>
#include <cstdlib>

// Thread-local storage for error messages
static thread_local std::string last_error;

// Internal database wrapper to hold both readable and writable database handles
struct XapianDatabase {
    Xapian::WritableDatabase db;
    std::string path;

    XapianDatabase(const std::string& p) : path(p), db(p, Xapian::DB_CREATE_OR_OPEN) {}
};

extern "C" {

xapian_db xapian_open(const char* path) {
    try {
        XapianDatabase* wrapper = new XapianDatabase(path);
        last_error.clear();
        return static_cast<xapian_db>(wrapper);
    } catch (const Xapian::Error& e) {
        last_error = e.get_description();
        return nullptr;
    } catch (const std::exception& e) {
        last_error = e.what();
        return nullptr;
    }
}

void xapian_close(xapian_db db) {
    if (db != nullptr) {
        XapianDatabase* wrapper = static_cast<XapianDatabase*>(db);
        try {
            wrapper->db.close();
        } catch (...) {
            // Ignore errors during close
        }
        delete wrapper;
    }
}

int xapian_index(xapian_db db, const char* chunk_id, const char* doc_id, const char* content) {
    if (db == nullptr || chunk_id == nullptr || content == nullptr) {
        last_error = "invalid arguments: db, chunk_id, and content must not be null";
        return -1;
    }

    try {
        XapianDatabase* wrapper = static_cast<XapianDatabase*>(db);

        // Create a term generator for indexing
        Xapian::TermGenerator indexer;
        indexer.set_stemmer(Xapian::Stem("en"));
        indexer.set_stemming_strategy(Xapian::TermGenerator::STEM_SOME);

        // Create a new document
        Xapian::Document doc;
        indexer.set_document(doc);

        // Index the content with positional information for phrase queries
        indexer.index_text(content);

        // Store metadata
        doc.add_value(0, chunk_id);  // Slot 0: chunk_id for retrieval
        if (doc_id != nullptr) {
            doc.add_value(1, doc_id);  // Slot 1: parent document ID
        }

        // Store the original content for potential snippeting
        doc.set_data(content);

        // Use chunk_id as the unique identifier term
        std::string id_term = "Q" + std::string(chunk_id);
        doc.add_boolean_term(id_term);

        // Replace or add the document
        wrapper->db.replace_document(id_term, doc);
        wrapper->db.commit();

        last_error.clear();
        return 0;
    } catch (const Xapian::Error& e) {
        last_error = e.get_description();
        return -1;
    } catch (const std::exception& e) {
        last_error = e.what();
        return -1;
    }
}

int xapian_delete(xapian_db db, const char* chunk_id) {
    if (db == nullptr || chunk_id == nullptr) {
        last_error = "invalid arguments: db and chunk_id must not be null";
        return -1;
    }

    try {
        XapianDatabase* wrapper = static_cast<XapianDatabase*>(db);

        std::string id_term = "Q" + std::string(chunk_id);
        wrapper->db.delete_document(id_term);
        wrapper->db.commit();

        last_error.clear();
        return 0;
    } catch (const Xapian::Error& e) {
        last_error = e.get_description();
        return -1;
    } catch (const std::exception& e) {
        last_error = e.what();
        return -1;
    }
}

SearchResults xapian_search(xapian_db db, const char* query_str, int limit) {
    SearchResults results = {nullptr, 0};

    if (db == nullptr || query_str == nullptr || limit <= 0) {
        last_error = "invalid arguments";
        return results;
    }

    try {
        XapianDatabase* wrapper = static_cast<XapianDatabase*>(db);

        // Create a query parser with database for proper stemming and case handling
        Xapian::QueryParser parser;
        parser.set_database(wrapper->db);
        parser.set_stemmer(Xapian::Stem("en"));
        parser.set_stemming_strategy(Xapian::QueryParser::STEM_SOME);
        parser.set_default_op(Xapian::Query::OP_OR);

        // Parse the query with partial matching for better recall
        Xapian::Query query = parser.parse_query(
            query_str,
            Xapian::QueryParser::FLAG_DEFAULT |
            Xapian::QueryParser::FLAG_WILDCARD |
            Xapian::QueryParser::FLAG_PARTIAL
        );

        // If empty query, return no results
        if (query.empty()) {
            last_error.clear();
            return results;
        }

        // Create an enquire object and run the query
        Xapian::Enquire enquire(wrapper->db);
        enquire.set_query(query);

        // Get the matching documents
        Xapian::MSet matches = enquire.get_mset(0, limit);

        if (matches.empty()) {
            last_error.clear();
            return results;
        }

        // Allocate results array
        results.count = static_cast<int>(matches.size());
        results.results = static_cast<SearchResult*>(
            malloc(sizeof(SearchResult) * results.count)
        );

        if (results.results == nullptr) {
            last_error = "memory allocation failed";
            results.count = 0;
            return results;
        }

        // Populate results
        int i = 0;
        for (Xapian::MSetIterator it = matches.begin(); it != matches.end(); ++it, ++i) {
            Xapian::Document doc = it.get_document();
            std::string chunk_id = doc.get_value(0);

            // Copy chunk_id (caller must free)
            results.results[i].chunk_id = strdup(chunk_id.c_str());

            // Normalize score to 0-1 range using MSet's max_possible
            double max_weight = matches.get_max_possible();
            if (max_weight > 0) {
                results.results[i].score = it.get_weight() / max_weight;
            } else {
                results.results[i].score = 0.0;
            }
        }

        last_error.clear();
        return results;
    } catch (const Xapian::Error& e) {
        last_error = e.get_description();
        // Clean up any partial results
        if (results.results != nullptr) {
            for (int i = 0; i < results.count; ++i) {
                free(results.results[i].chunk_id);
            }
            free(results.results);
        }
        results.results = nullptr;
        results.count = 0;
        return results;
    } catch (const std::exception& e) {
        last_error = e.what();
        if (results.results != nullptr) {
            for (int i = 0; i < results.count; ++i) {
                free(results.results[i].chunk_id);
            }
            free(results.results);
        }
        results.results = nullptr;
        results.count = 0;
        return results;
    }
}

void xapian_free_results(SearchResults results) {
    if (results.results != nullptr) {
        for (int i = 0; i < results.count; ++i) {
            free(results.results[i].chunk_id);
        }
        free(results.results);
    }
}

const char* xapian_get_error(void) {
    return last_error.c_str();
}

} // extern "C"
