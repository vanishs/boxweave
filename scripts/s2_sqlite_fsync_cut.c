// One-use Linux LD_PRELOAD failpoint: kill a nonroot test process on the
// actual SQLite WAL fsync/fdatasync after one original NativeAck begins.
// Never linked into or shipped with BoxWeave. Intended ONLY for isolated CI.
#define _GNU_SOURCE
#include <dlfcn.h>
#include <fcntl.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <unistd.h>

static int should_cut(int fd) {
    const char *dbwal = getenv("BW_S2_FSYNC_DB_WAL");
    const char *arm = getenv("BW_S2_FSYNC_ARM_FILE");
    const char *witness = getenv("BW_S2_FSYNC_WITNESS_FILE");
    const char *mode = getenv("BW_S2_FSYNC_CUT_MODE");
    const char *gh = getenv("GITHUB_ACTIONS");
    const char *runner = getenv("RUNNER_ENVIRONMENT");
    if (!dbwal || !arm || !witness || !mode || !gh || !runner ||
        strcmp(gh, "true") || strcmp(runner, "github-hosted") ||
        strncmp(dbwal, "/tmp/bw-v1-s2-ghci.", 19) ||
        (!strcmp(mode, "before") == !strcmp(mode, "after"))) {
        return 0;
    }
    if (access(arm, F_OK) != 0) {
        return 0;
    }
    char fdpath[64], actual[4096];
    snprintf(fdpath, sizeof(fdpath), "/proc/self/fd/%d", fd);
    ssize_t n = readlink(fdpath, actual, sizeof(actual) - 1);
    if (n < 0) {
        return 0;
    }
    actual[n] = '\0';
    return strcmp(dbwal, actual) == 0;
}

static void cut_now(void) {
    const char *witness = getenv("BW_S2_FSYNC_WITNESS_FILE");
    int f = open(witness, O_WRONLY | O_CREAT | O_EXCL | O_CLOEXEC, 0600);
    if (f < 0) {
        _exit(91);
    }
    static const char msg[] = "ORIGINAL_SQLITE_WAL_FSYNC_CUT_ACTUALLY_TRIGGERED\n";
    (void)write(f, msg, sizeof(msg) - 1);
    (void)close(f);
    (void)kill(getpid(), SIGKILL);
    _exit(92);
}

int fdatasync(int fd) {
    static int (*real_fn)(int) = NULL;
    if (!real_fn) {
        real_fn = dlsym(RTLD_NEXT, "fdatasync");
    }
    if (!real_fn) {
        _exit(93);
    }
    if (!should_cut(fd)) {
        return real_fn(fd);
    }
    const char *mode = getenv("BW_S2_FSYNC_CUT_MODE");
    if (!strcmp(mode, "after") && real_fn(fd) != 0) {
        return -1;
    }
    cut_now();
    return -1;
}

int fsync(int fd) {
    static int (*real_fn)(int) = NULL;
    if (!real_fn) {
        real_fn = dlsym(RTLD_NEXT, "fsync");
    }
    if (!real_fn) {
        _exit(94);
    }
    if (!should_cut(fd)) {
        return real_fn(fd);
    }
    const char *mode = getenv("BW_S2_FSYNC_CUT_MODE");
    if (!strcmp(mode, "after") && real_fn(fd) != 0) {
        return -1;
    }
    cut_now();
    return -1;
}
