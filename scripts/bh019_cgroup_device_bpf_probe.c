// One-use GitHub runner only: actual BPF_PROG_LOAD permission probe for
// BPF_PROG_TYPE_CGROUP_DEVICE. NO program is ever attached to any cgroup,
// device or namespace. This is NOT a security enforcement or nesting PASS.
#define _GNU_SOURCE
#include <errno.h>
#include <linux/bpf.h>
#include <linux/filter.h>
#include <stdint.h>
#include <stdio.h>
#include <sys/syscall.h>
#include <unistd.h>

int main(void) {
    static const char license[] = "GPL";
    struct bpf_insn instructions[2] = {
        {.code = BPF_ALU64 | BPF_MOV | BPF_K, .dst_reg = BPF_REG_0, .imm = 1},
        {.code = BPF_JMP | BPF_EXIT},
    };
    union bpf_attr attributes = {0};
    attributes.prog_type = BPF_PROG_TYPE_CGROUP_DEVICE;
    attributes.insn_cnt = 2;
    attributes.insns = (uint64_t)(uintptr_t)instructions;
    attributes.license = (uint64_t)(uintptr_t)license;
    int program = (int)syscall(SYS_bpf, BPF_PROG_LOAD, &attributes, sizeof(attributes));
    if (program >= 0) {
        close(program);
        puts("BH019_CGROUP_DEVICE_BPF_PROG_LOAD=GRANTED_NO_ATTACH");
        return 0;
    }
    printf("BH019_CGROUP_DEVICE_BPF_PROG_LOAD=DENIED errno=%d\n", errno);
    return 0;
}
