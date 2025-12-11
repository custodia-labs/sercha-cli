/*
 * xapian_wrapper.h - C-compatible wrapper for Xapian C++ API
 *
 * This header provides a C interface to Xapian for use with CGO.
 * All functions use C types to ensure compatibility with Go.
 */

#ifndef XAPIAN_WRAPPER_H
#define XAPIAN_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

/* Opaque handle to Xapian database */
typedef void* xapian_db;

/*
 * xapian_open - Open or create a Xapian database
 *
 * @param path: Directory path for the database
 * @return: Database handle, or NULL on error
 */
xapian_db xapian_open(const char* path);

/*
 * xapian_close - Close a Xapian database
 *
 * @param db: Database handle
 */
void xapian_close(xapian_db db);

/*
 * xapian_index - Add or update a document in the index
 *
 * @param db: Database handle
 * @param chunk_id: Unique identifier for the chunk
 * @param doc_id: Parent document ID
 * @param content: Text content to index
 * @return: 0 on success, -1 on error
 */
int xapian_index(xapian_db db, const char* chunk_id, const char* doc_id, const char* content);

/*
 * xapian_delete - Remove a document from the index
 *
 * @param db: Database handle
 * @param chunk_id: Unique identifier for the chunk to delete
 * @return: 0 on success, -1 on error
 */
int xapian_delete(xapian_db db, const char* chunk_id);

/*
 * SearchResult - Single search result
 */
typedef struct {
    char* chunk_id;
    double score;
} SearchResult;

/*
 * SearchResults - Array of search results
 */
typedef struct {
    SearchResult* results;
    int count;
} SearchResults;

/*
 * xapian_search - Perform a search query
 *
 * @param db: Database handle
 * @param query: Search query string
 * @param limit: Maximum number of results
 * @return: SearchResults struct (caller must free with xapian_free_results)
 */
SearchResults xapian_search(xapian_db db, const char* query, int limit);

/*
 * xapian_free_results - Free search results memory
 *
 * @param results: SearchResults to free
 */
void xapian_free_results(SearchResults results);

/*
 * xapian_get_error - Get the last error message
 *
 * @return: Error message string (valid until next xapian call)
 */
const char* xapian_get_error(void);

#ifdef __cplusplus
}
#endif

#endif /* XAPIAN_WRAPPER_H */
