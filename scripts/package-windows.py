#!/usr/bin/env python3
"""Build and verify a native Windows portable ZIP without installing anything."""

import argparse
import hashlib
import os
import pathlib
import subprocess
import sys
import tempfile
import zipfile

ROOT = pathlib.Path(__file__).resolve().parent.parent


def command(*args):
    return subprocess.check_output(args, cwd=ROOT, text=True, encoding="utf-8").strip()


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", type=pathlib.Path, default=ROOT / "build" / "releases")
    args = parser.parse_args()
    if sys.platform != "win32":
        parser.error("run on Windows: the package is verified by executing its binary")
    out = args.out.resolve()
    out.mkdir(parents=True, exist_ok=True)
    revision = command("git", "rev-parse", "HEAD")
    arch = command("go", "env", "GOARCH")

    with tempfile.TemporaryDirectory(prefix="nekophage-package-", dir=out) as temporary:
        work = pathlib.Path(temporary)
        stage = work / "Nekophage"
        stage.mkdir()
        exe = stage / "Nekophage.exe"
        subprocess.run(["go", "build", "-trimpath", "-o", str(exe), "."], cwd=ROOT, check=True)
        info = command(str(exe), "-version")
        version = info.splitlines()[0].split()[1]
        if f"commit {revision}" not in info:
            raise RuntimeError("the binary does not report the checked-out commit")
        (stage / "LICENSE").write_bytes((ROOT / "LICENSE").read_bytes())
        (stage / "BUILD-INFO.txt").write_text(
            info + "\n\nUnsigned local test build. Command: go build -trimpath .\n",
            encoding="utf-8",
        )
        (stage / "README.zh.txt").write_text(
            "Nekophage Windows 便携测试包\n\n"
            "解压整个目录,运行 Nekophage.exe。无需安装 Go 或单独启动 Core。\n"
            "阅读器和网页登录需要 Microsoft Edge WebView2 运行时。\n"
            "初始不附带第三方漫画源;在漫画源管理中添加自己的源或导入备份。\n"
            "数据保存在 %APPDATA%\\Nekophage,升级时无需复制数据到安装目录。\n"
            "AI 功能连接另行安装的本机 Magpie;此包不包含 Magpie。\n\n"
            "这是未签名的本地测试包,不是 Windows 安装程序。\n"
            "版本、Core 和提交见 BUILD-INFO.txt;文件校验见 SHA256SUMS.txt。\n",
            encoding="utf-8",
        )
        (stage / "README.en.txt").write_text(
            "Nekophage Windows portable test build\n\n"
            "Extract the whole directory and run Nekophage.exe. Go and a separate Core process are unnecessary.\n"
            "The reader and web sign-in require Microsoft Edge WebView2 Runtime.\n"
            "No third-party sources are bundled; add your sources or import a backup in the source manager.\n"
            "Data lives in %APPDATA%\\Nekophage and does not need copying into the application directory.\n"
            "AI connects to a separately installed local Magpie; Magpie is not included.\n\n"
            "This is an unsigned local test build, not a Windows installer.\n"
            "See BUILD-INFO.txt for versions and the commit, and SHA256SUMS.txt for file hashes.\n",
            encoding="utf-8",
        )
        files = sorted(stage.iterdir())
        (stage / "SHA256SUMS.txt").write_text(
            "".join(f"{digest(path)}  {path.name}\n" for path in files), encoding="utf-8"
        )
        name = f"Nekophage-{version}-windows-{arch}-{revision[:7]}.zip"
        archive = work / name
        with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED) as bundle:
            for path in sorted(stage.iterdir()):
                bundle.write(path, f"Nekophage/{path.name}")
        unpacked = work / "unpacked"
        with zipfile.ZipFile(archive) as bundle:
            if bundle.testzip() is not None:
                raise RuntimeError("ZIP CRC verification failed")
            bundle.extractall(unpacked)
        unpacked_stage = unpacked / "Nekophage"
        for line in (unpacked_stage / "SHA256SUMS.txt").read_text(encoding="utf-8").splitlines():
            expected, filename = line.split("  ", 1)
            if digest(unpacked_stage / filename) != expected:
                raise RuntimeError(f"unpacked checksum mismatch: {filename}")
        if command(str(unpacked_stage / "Nekophage.exe"), "-version") != info:
            raise RuntimeError("the unpacked executable reports different build information")
        archive_hash = digest(archive)
        destination = out / name
        os.replace(archive, destination)
        destination.with_suffix(".zip.sha256").write_text(
            f"{archive_hash}  {name}\n", encoding="utf-8"
        )
        print(f"PASS Windows build, ZIP CRC, unpacked checksums and executable -version")
        print(f"package: {destination}")
        print(f"bytes: {destination.stat().st_size}")
        print(f"sha256: {archive_hash}")
        print(info)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
