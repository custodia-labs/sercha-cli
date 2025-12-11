#ifndef SERCHA_XAPIAN_WRAPPER_H
#define SERCHA_XAPIAN_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stddef.h>

// Opaque handle for the Xapian database.
typedef struct XapianDatabase XapianDatabase;

// SearchResult holds a single search result.
typedef struct {
    char* chunk_id;
    double score;
} XapianSearchResult;

// Open or create a Xapian database.
// Returns NULL on error.
XapianDatabase* xapian_open(const char* path);

// Index a document.
// Returns 0 on success, -1 on error.
int xapian_index(XapianDatabase* db, const char* chunk_id, const char* content);

// Delete a document.
// Returns 0 on success, -1 on error.
int xapian_delete(XapianDatabase* db, const char* chunk_id);

// Search for documents.
// Returns the number of results, or -1 on error.
int xapian_search(XapianDatabase* db, const char* query, int limit,
                  XapianSearchResult** results);

// Free search results.
void xapian_free_results(XapianSearchResult* results, int count);

// Close and free the database.
void xapian_close(XapianDatabase* db);

#ifdef __cplusplus
}
#endif

#endif // SERCHA_XAPIAN_WRAPPER_H
