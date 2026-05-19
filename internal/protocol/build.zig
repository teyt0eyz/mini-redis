const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const lib = b.addLibrary(.{
        .name = "protocol",
        .linkage = .dynamic,
        .root_module = b.createModule(.{
            .root_source_file = b.path("main.zig"),
            .target = target,
            .optimize = optimize,
            .link_libc = true,
        }),
    });

    // ใน Zig 0.16.0 addObjectFile ย้ายมาอยู่ใน root_module
    lib.root_module.addObjectFile(.{ .cwd_relative = "../storage/target/release/librust_storage.a" });

    b.installArtifact(lib);
}
