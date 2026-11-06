#!/usr/bin/env python3
"""Build/verify a Windows ZIP and optionally a per-user NSIS installer."""

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
    parser.add_argument("--nsis", type=pathlib.Path, help="makensis.exe path; also build a per-user setup executable")
    args = parser.parse_args()
    if sys.platform != "win32":
        parser.error("run on Windows: the package is verified by executing its binary")
    # A branch switch can leave go.work and corestamp.go on an older Core.
    # Reject that checkout before building or publishing any package files.
    subprocess.run(["sh", "scripts/check-core-pin.sh"], cwd=ROOT, check=True)
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
        # LICENSE references the LGPL's incorporated GPL text, and
        # LGPL 4(b) requires that text to travel with the binary.
        (stage / "COPYING").write_bytes((ROOT / "COPYING").read_bytes())
        assets = ROOT / "internal" / "player" / "assets"
        for source, name in [
            ("HLS-LICENSE.txt", "HLS-LICENSE.txt"),
            ("APACHE-2.0.txt", "HLS-APACHE-2.0.txt"),
            ("NOTICE.md", "HLS-NOTICE.zh.txt"),
            ("NOTICE.en.md", "HLS-NOTICE.en.txt"),
        ]:
            (stage / name).write_bytes((assets / source).read_bytes())
        (stage / "BUILD-INFO.txt").write_text(
            info + "\n\nUnsigned local test build. Command: go build -trimpath .\n",
            encoding="utf-8",
        )
        (stage / "README.zh.txt").write_text(
            "Nekophage Windows 使用说明\n\n"
            "安装版运行安装后的 Nekophage.exe;便携版解压整个目录后运行。无需安装 Go 或单独启动 Core。\n"
            "阅读器、播放器和网页登录需要 Microsoft Edge WebView2 运行时。\n"
            "初始不附带第三方源;在源管理选择漫画源、视频源或 Kazumi 规则,从文件/链接安装。\n"
            "视频包含本机媒体代理、HLS.js 与进度续播;具体平台与媒体格式验证见仓库状态记录。\n"
            "数据保存在 %APPDATA%\\Nekophage,升级时无需复制数据到安装目录。\n"
            "AI 功能连接另行安装的本机 Magpie;此包不包含 Magpie。\n\n"
            "这是未签名的本地测试构建。安装版仅为当前用户安装,卸载保留 APPDATA 中的资料。\n"
            "版本、Core 和提交见 BUILD-INFO.txt;文件校验见 SHA256SUMS.txt。\n",
            encoding="utf-8",
        )
        (stage / "README.en.txt").write_text(
            "Nekophage Windows instructions\n\n"
            "Run the installed Nekophage.exe, or extract the entire portable ZIP before running. Go and a separate Core process are unnecessary.\n"
            "The reader, player and web sign-in require Microsoft Edge WebView2 Runtime.\n"
            "No third-party sources are bundled; select comic/video/Kazumi in source management and install a file or URL.\n"
            "Video includes a local media proxy, HLS.js and persisted resume; see the repository status for platform/format evidence.\n"
            "Data lives in %APPDATA%\\Nekophage and does not need copying into the application directory.\n"
            "AI connects to a separately installed local Magpie; Magpie is not included.\n\n"
            "This is an unsigned local test build. Setup installs for the current user; uninstall preserves APPDATA profile files.\n"
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
        if args.nsis:
            nsis = args.nsis.resolve()
            installer_name = f"Nekophage-{version}-windows-{arch}-{revision[:7]}-setup.exe"
            installer = work / installer_name
            subprocess.run(
                [str(nsis), "/V2", f"/DPAYLOAD_DIR={stage}",
                 f"/DOUTPUT_FILE={installer}", f"/DAPP_VERSION={version}",
                 str(ROOT / "scripts" / "installer-windows.nsi")],
                cwd=ROOT, check=True,
            )
            installer_hash = digest(installer)
            installer_destination = out / installer_name
            os.replace(installer, installer_destination)
            installer_destination.with_suffix(".exe.sha256").write_text(
                f"{installer_hash}  {installer_name}\n", encoding="utf-8"
            )
            print(f"PASS NSIS installer build (execution tested separately)")
            print(f"installer: {installer_destination}")
            print(f"installer bytes: {installer_destination.stat().st_size}")
            print(f"installer sha256: {installer_hash}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
