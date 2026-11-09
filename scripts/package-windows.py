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
            "便携版:把整个 Nekophage 目录解压到可写的位置(例如 D:\\Apps\\Nekophage),运行 Nekophage.exe。无需安装 Go 或单独启动 Core。\n"
            "阅读器、播放器和网页登录需要 Microsoft Edge WebView2 运行时。\n"
            "初始不附带第三方源;在源管理选择漫画源、视频源或 Kazumi 规则,从文件/链接安装。\n"
            "视频包含本机媒体代理、HLS.js 与进度续播;具体平台与媒体格式验证见仓库状态记录。\n"
            "AI 功能连接另行安装的本机 Magpie;此包不包含 Magpie。\n\n"
            "所有资料都放在程序目录里,程序根目录只有随包文件:\n"
            "  data\\     源、插件、服务、Cookie、书架与设置\n"
            "  cache\\    可以随时删除、会重新生成的内容\n"
            "  logs\\     诊断日志\n"
            "  webview\\  阅读器与播放器的 WebView2 配置\n"
            "升级:用新版本的文件覆盖旧目录即可,这四个目录保持不动。删除整个目录即彻底移除,不在用户目录留下资料。\n"
            "首次启动时,如果 data 为空而 %APPDATA%\\Nekophage 里有旧资料,会复制一份过来,原资料不改动。\n"
            "程序目录不可写(如放在 Program Files 下)时,改用 %APPDATA%\\Nekophage。\n\n"
            "这是未签名的本地测试构建。版本、Core 和提交见 BUILD-INFO.txt;文件校验见 SHA256SUMS.txt。\n",
            encoding="utf-8",
        )
        (stage / "README.en.txt").write_text(
            "Portable: extract the whole Nekophage folder somewhere writable (for example D:\\Apps\\Nekophage) and run Nekophage.exe. Go and a separate Core process are unnecessary.\n"
            "The reader, player and web sign-in require Microsoft Edge WebView2 Runtime.\n"
            "No third-party sources are bundled; select comic/video/Kazumi in source management and install a file or URL.\n"
            "Video includes a local media proxy, HLS.js and persisted resume; see the repository status for platform/format evidence.\n"
            "AI connects to a separately installed local Magpie; Magpie is not included.\n\n"
            "Everything the app keeps stays in its folder, and the folder's top level holds only the shipped files:\n"
            "  data\\     sources, plugins, services, cookies, shelf and settings\n"
            "  cache\\    anything that can be deleted and rebuilt\n"
            "  logs\\     diagnostics\n"
            "  webview\\  the WebView2 profile of the reader and player\n"
            "To upgrade, copy the new files over the old folder; the four folders stay as they are. Deleting the folder removes the app completely and leaves nothing in the user profile.\n"
            "On first start, if data is empty and %APPDATA%\\Nekophage holds an earlier profile, it is copied in and the original is left untouched.\n"
            "When the folder cannot be written (for example under Program Files), %APPDATA%\\Nekophage is used instead.\n\n"
            "This is an unsigned local test build. See BUILD-INFO.txt for versions and the commit, and SHA256SUMS.txt for file hashes.\n",
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
        # newline="\n": on Windows write_text turns \n into \r\n, and sha256sum -c
        # then reads the \r as part of the file name and cannot open the file.
        # The checksum is published next to the package, so it has to verify
        # with the stock tool on every platform.
        destination.with_suffix(".zip.sha256").write_text(
            f"{archive_hash}  {name}\n", encoding="utf-8", newline="\n"
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
                f"{installer_hash}  {installer_name}\n", encoding="utf-8", newline="\n"
            )
            print(f"PASS NSIS installer build (execution tested separately)")
            print(f"installer: {installer_destination}")
            print(f"installer bytes: {installer_destination.stat().st_size}")
            print(f"installer sha256: {installer_hash}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
