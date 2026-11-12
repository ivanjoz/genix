// nosync_preload: the part of libeatmydata the OceanBase benchmarks rely on, buildable for an older
// glibc than the host's (the seekdb image runs Rocky 9, glibc 2.34; Fedora's libeatmydata needs
// newer ABI tags). Strips O_SYNC/O_DSYNC from open() and turns fsync/fdatasync/syncfs/sync_file_range
// into no-ops, so a redo write costs what it costs on a server SSD with power-loss protection.
//
// NOT DURABLE: for load tests only. Build (see seekdb_fast_redo in run_seekdb.sh):
//   gcc -O2 -shared -fPIC -Wl,-z,nopack-relative-relocs -o nosync_preload.so nosync_preload.c -ldl
#define _GNU_SOURCE
#include <dlfcn.h>
#include <fcntl.h>
#include <stdarg.h>
#include <sys/types.h>

#define SYNC_FLAGS (O_SYNC | O_DSYNC)

typedef int (*open_function)(const char *, int, ...);
typedef int (*openat_function)(int, const char *, int, ...);

static mode_t mode_argument(int flags, va_list arguments) {
	return (flags & (O_CREAT | O_TMPFILE)) ? (mode_t)va_arg(arguments, int) : 0;
}

#define DEFINE_OPEN(name)                                                     \
	int name(const char *path, int flags, ...) {                              \
		static open_function real_open;                                       \
		if (!real_open) real_open = (open_function)dlsym(RTLD_NEXT, #name);   \
		va_list arguments;                                                    \
		va_start(arguments, flags);                                           \
		mode_t mode = mode_argument(flags, arguments);                        \
		va_end(arguments);                                                    \
		return real_open(path, flags & ~SYNC_FLAGS, mode);                    \
	}

#define DEFINE_OPENAT(name)                                                       \
	int name(int directory, const char *path, int flags, ...) {                   \
		static openat_function real_openat;                                       \
		if (!real_openat) real_openat = (openat_function)dlsym(RTLD_NEXT, #name); \
		va_list arguments;                                                        \
		va_start(arguments, flags);                                               \
		mode_t mode = mode_argument(flags, arguments);                            \
		va_end(arguments);                                                        \
		return real_openat(directory, path, flags & ~SYNC_FLAGS, mode);           \
	}

DEFINE_OPEN(open)
DEFINE_OPEN(open64)
DEFINE_OPENAT(openat)
DEFINE_OPENAT(openat64)

int fsync(int fd) { (void)fd; return 0; }
int fdatasync(int fd) { (void)fd; return 0; }
int syncfs(int fd) { (void)fd; return 0; }
int sync_file_range(int fd, off_t offset, off_t length, unsigned int flags) {
	(void)fd; (void)offset; (void)length; (void)flags;
	return 0;
}
