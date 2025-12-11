#include "xapian_wrapper.h"

// Stub implementation - will be filled in when implementing functionality

extern "C" {

XapianDatabase* xapian_open(const char* path) {
    (void)path;
    return nullptr;
}

int xapian_index(XapianDatabase* db, const char* chunk_id, const char* content) {
    (void)db;
    (void)chunk_id;
    (void)content;
    return -1;
}

int xapian_delete(XapianDatabase* db, const char* chunk_id) {
    (void)db;
    (void)chunk_id;
    return -1;
}

int xapian_search(XapianDatabase* db, const char* query, int limit,
                  XapianSearchResult** results) {
    (void)db;
    (void)query;
    (void)limit;
    (void)results;
    return -1;
}

void xapian_free_results(XapianSearchResult* results, int count) {
    (void)results;
    (void)count;
}

void xapian_close(XapianDatabase* db) {
    (void)db;
}

} // extern "C"
