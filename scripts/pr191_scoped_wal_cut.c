// Test-only installed Scoped WAL commit witness. Never linked into BoxWeave.
// Reuse the existing fsync-cut approach, but arm only on a unique marker in
// a dedicated page of the exact transaction, not the next arbitrary sync.
#define _GNU_SOURCE
#include <dlfcn.h>
#include <fcntl.h>
#include <signal.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

#define WAL "/var/lib/boxweave/ledger.sqlite-wal"
#define ARM "/var/lib/boxweave/p1b-wal-cut-arm"
#define WITNESS "/var/lib/boxweave/p1b-wal-cut-witness"

static uint32_t be32(const unsigned char *p) {
    return ((uint32_t)p[0] << 24) | ((uint32_t)p[1] << 16) |
           ((uint32_t)p[2] << 8) | (uint32_t)p[3];
}

// Read-only, bounded observer, also exercised on real private SQLite WAL
// files in the source test. The live fixture checkpoints before arming and
// writes the nonce only in the dedicated marker table's first/only row.
int p1b_wal_has_marker(int fd, uint32_t page, const char *nonce) {
    unsigned char header[32], frame[24], data[65536];
    struct stat st;
    if (page == 0 || !nonce || strlen(nonce) != 32 ||
        fstat(fd, &st) != 0 || st.st_size < 32 || st.st_size > 64 * 1024 * 1024 ||
        pread(fd, header, sizeof(header), 0) != (ssize_t)sizeof(header)) {
        return 0;
    }
    uint32_t magic = be32(header), size = be32(header + 8);
    if ((magic != 0x377f0682 && magic != 0x377f0683) ||
        be32(header + 4) != 3007000 ||
        size < 512 || size > sizeof(data) || (size & (size - 1))) {
        return 0;
    }
    int seen = 0;
    for (off_t off = 32; off + 24 + size <= st.st_size; off += 24 + size) {
        if (pread(fd, frame, sizeof(frame), off) != (ssize_t)sizeof(frame) ||
            memcmp(frame + 8, header + 16, 8) || be32(frame) == 0) {
            break;  // stale or incomplete WAL generation
        }
        if (be32(frame) == page) {
            if (pread(fd, data, size, off + 24) != (ssize_t)size) {
                return 0;
            }
            seen = memmem(data, size, nonce, 32) != NULL;
        }
        if (be32(frame + 4) != 0) {
            if (seen) {
                return 1;
            }
            seen = 0;
        }
    }
    return 0;
}

static int should_cut(int fd) {
    const char *gh = getenv("GITHUB_ACTIONS"), *runner = getenv("RUNNER_ENVIRONMENT");
    const char *mode = getenv("BW_P1B_WAL_PHASE"), *nonce = getenv("BW_P1B_WAL_NONCE");
    if (geteuid() == 0 || !gh || strcmp(gh, "true") || !runner ||
        strcmp(runner, "github-hosted") || !mode ||
        (strcmp(mode, "before") && strcmp(mode, "after")) || !nonce) {
        return 0;
    }
    // Arm only after the clean controller starts and the fixture creates
    // its dedicated marker page. A missing arm is an ordinary sync.
    char pg[32], *end;
    int arm = open(ARM, O_RDONLY | O_CLOEXEC | O_NOFOLLOW);
    if (arm < 0) {
        return 0;
    }
    ssize_t len = read(arm, pg, sizeof(pg) - 1);
    (void)close(arm);
    if (len <= 0 || len >= (ssize_t)sizeof(pg) - 1) {
        return 0;
    }
    pg[len] = '\0';
    unsigned long page = strtoul(pg, &end, 10);
    if (!*pg || *end || page == 0 || page > UINT32_MAX) {
        return 0;
    }
    char link[64], actual[256];
    snprintf(link, sizeof(link), "/proc/self/fd/%d", fd);
    ssize_t n = readlink(link, actual, sizeof(actual) - 1);
    if (n < 0) {
        return 0;
    }
    actual[n] = '\0';
    return !strcmp(actual, WAL) && p1b_wal_has_marker(fd, (uint32_t)page, nonce);
}

static void cut(void) {
    int fd = open(WITNESS, O_WRONLY | O_CREAT | O_EXCL | O_CLOEXEC | O_NOFOLLOW, 0600);
    if (fd < 0) {
        _exit(91);
    }
    char msg[128];
    int n = snprintf(msg, sizeof(msg), "P1B_EXACT_WAL_COMMIT_SYNC_CUT %s %ld\n",
                     getenv("BW_P1B_WAL_PHASE"), (long)getpid());
    if (n <= 0 || (size_t)n >= sizeof(msg) || write(fd, msg, (size_t)n) != n || close(fd) != 0) {
        _exit(92);
    }
    (void)kill(getpid(), SIGKILL);
    _exit(93);
}

static int sync_cut(int fd, const char *symbol) {
    int (*real_fn)(int) = dlsym(RTLD_NEXT, symbol);
    if (!real_fn) {
        _exit(94);
    }
    if (!should_cut(fd)) {
        return real_fn(fd);
    }
    if (!strcmp(getenv("BW_P1B_WAL_PHASE"), "after") && real_fn(fd) != 0) {
        return -1;
    }
    cut();
    return -1;
}

int fsync(int fd) { return sync_cut(fd, "fsync"); }
int fdatasync(int fd) { return sync_cut(fd, "fdatasync"); }
