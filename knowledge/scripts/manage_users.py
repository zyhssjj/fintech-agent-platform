"""Command-line account bootstrap and emergency password reset utility."""

import argparse
import getpass
import sys
from pathlib import Path


PROJECT_DIR = Path(__file__).resolve().parent.parent
if str(PROJECT_DIR) not in sys.path:
    sys.path.insert(0, str(PROJECT_DIR))

from scripts.auth_store import ALL_DOMAINS, AuthStore  # noqa: E402


def requested_password(args, confirmation=True):
    password = args.password or getpass.getpass("密码（至少8个字符）: ")
    if confirmation and not args.password:
        repeated = getpass.getpass("再次输入密码: ")
        if password != repeated:
            raise ValueError("两次输入的密码不一致")
    return password


def bootstrap(args, store):
    if store.has_superadmin():
        raise ValueError("超级管理员已经存在，系统只允许一个超级管理员")
    user = store.create_user(
        args.name,
        args.position,
        args.username,
        requested_password(args),
        role="superadmin",
        domains=ALL_DOMAINS,
        must_change_password=True,
    )
    print("超级管理员已创建：{0}（{1}）".format(user["username"], user["name"]))
    print("首次登录后必须修改初始密码。")


def reset_superadmin(args, store):
    row = store.get_user_by_username(args.username)
    if not row or row["role"] != "superadmin":
        raise ValueError("未找到该超级管理员账号")
    current = store.get_user(row["id"])
    store.update_user(
        current["id"],
        name=current["name"],
        position=current["position"],
        username=current["username"],
        role=current["role"],
        domains=current["domains"],
        password=requested_password(args),
        must_change_password=True,
        actor_user_id=current["id"],
    )
    print("超级管理员密码已重置，所有旧会话已经失效。")


def build_parser():
    parser = argparse.ArgumentParser(description="迎风聚智知识库用户管理工具")
    parser.add_argument(
        "--runtime-dir",
        default=str(PROJECT_DIR / "runtime"),
        help="系统 runtime 目录",
    )
    commands = parser.add_subparsers(dest="command", required=True)

    bootstrap_parser = commands.add_parser(
        "bootstrap-superadmin",
        help="首次创建唯一超级管理员",
    )
    bootstrap_parser.add_argument("--name", required=True)
    bootstrap_parser.add_argument("--position", required=True)
    bootstrap_parser.add_argument("--username", required=True)
    bootstrap_parser.add_argument("--password", help=argparse.SUPPRESS)
    bootstrap_parser.set_defaults(handler=bootstrap)

    reset_parser = commands.add_parser(
        "reset-superadmin-password",
        help="紧急重置超级管理员密码",
    )
    reset_parser.add_argument("--username", required=True)
    reset_parser.add_argument("--password", help=argparse.SUPPRESS)
    reset_parser.set_defaults(handler=reset_superadmin)
    return parser


def main():
    parser = build_parser()
    args = parser.parse_args()
    try:
        args.handler(args, AuthStore(args.runtime_dir))
    except (ValueError, KeyError) as error:
        parser.error(str(error))


if __name__ == "__main__":
    main()
