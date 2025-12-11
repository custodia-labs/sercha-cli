cmake_minimum_required(VERSION 3.20)
project(sercha-clib VERSION 0.1.0 LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# FetchContent for HNSWlib
include(FetchContent)

FetchContent_Declare(
    hnswlib
    GIT_REPOSITORY https://github.com/nmslib/hnswlib.git
    GIT_TAG v0.8.0
)
FetchContent_MakeAvailable(hnswlib)

# Find Xapian (system install)
find_package(PkgConfig REQUIRED)
pkg_check_modules(XAPIAN REQUIRED xapian-core)

# HNSWlib wrapper library
add_library(sercha_hnsw STATIC
    hnsw/hnsw_wrapper.cpp
)
target_include_directories(sercha_hnsw PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/hnsw
    ${hnswlib_SOURCE_DIR}
)
target_compile_options(sercha_hnsw PRIVATE -O3)

# Xapian wrapper library
add_library(sercha_xapian STATIC
    xapian/xapian_wrapper.cpp
)
target_include_directories(sercha_xapian PUBLIC
    ${CMAKE_CURRENT_SOURCE_DIR}/xapian
    ${XAPIAN_INCLUDE_DIRS}
)
target_link_libraries(sercha_xapian PRIVATE ${XAPIAN_LIBRARIES})
target_compile_options(sercha_xapian PRIVATE ${XAPIAN_CFLAGS_OTHER})

# Install targets
install(TARGETS sercha_hnsw sercha_xapian
    ARCHIVE DESTINATION lib
)
install(FILES
    hnsw/hnsw_wrapper.h
    xapian/xapian_wrapper.h
    DESTINATION include
)
