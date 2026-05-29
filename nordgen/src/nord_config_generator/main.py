import argparse
import asyncio
import re
import sys
import time

from .client import NordClient
from .constants import ALIAS_TO_GROUP_ID
from .generator import Generator
from .models import UserPreferences
from .ui import ConsoleManager

_TOKEN_PATTERN = re.compile(r"[0-9a-fA-F]{64}")

def _build_argument_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="nordgen", description="NordVPN WireGuard Config Generator"
    )
    
    parser.add_argument("-t", "--token", help="NordVPN Access Token")
    parser.add_argument("-d", "--dns", default="103.86.96.100", help="DNS Server")
    parser.add_argument("-i", "--ip", action="store_true", help="Use IP Endpoint")
    parser.add_argument("-k", "--keepalive", type=int, default=25, help="Keepalive seconds")
    parser.add_argument(
        "-g",
        "--group",
        nargs="*",
        choices=sorted(ALIAS_TO_GROUP_ID.keys()),
        help="Server groups to include. Example: -g standard p2p.",
    )
    parser.add_argument(
        "-e",
        "--exclude-dedicated",
        action="store_true",
        help="Exclude servers in the dedicated IP group",
    )

    subparsers = parser.add_subparsers(dest="command", title="Commands", metavar="<command>")
    
    keyp = subparsers.add_parser("get-key", help="Retrieve NordLynx private key")
    keyp.add_argument("-t", "--token", help="NordVPN Access Token", default="")

    return parser

async def _resolve_private_key(
    ui: ConsoleManager, client: NordClient, token: str
) -> str:
    if not token:
        token = ui.prompt_secret("Please enter your NordVPN access token")
    if _TOKEN_PATTERN.fullmatch(token) is None:
        ui.error("Invalid token format")
        return ""
    with ui.status("Validating token..."):
        key = await client.get_key(token)
    if not key:
        ui.fail("Token invalid")
        return ""
    ui.success("Token validated")
    return key

async def _run_get_key(ui: ConsoleManager, client: NordClient, token: str) -> None:
    ui.clear()
    ui.header()
    key = await _resolve_private_key(ui, client, token)
    if key:
        ui.show_key(key)
        
    if not token:
        ui.wait()

async def _run_generate(
    ui: ConsoleManager,
    client: NordClient,
    token: str,
    preferences: UserPreferences,
) -> None:
    is_interactive = not bool(token)

    ui.clear()
    ui.header()
    key = await _resolve_private_key(ui, client, token)
    if not key:
        if is_interactive:
            ui.wait()
        return

    if is_interactive:
        ui.clear()
        ui.header()
        preferences = ui.prompt_preferences(preferences)

    if preferences.keepalive < 0:
        ui.fail("Keepalive value must be greater than or equal to 0")
        if is_interactive:
            ui.wait()
        return

    if preferences.exclude_dedicated and preferences.groups and ALIAS_TO_GROUP_ID["dedicated"] in preferences.groups:
        ui.fail("Conflict: Cannot require 'dedicated' group while using exclude-dedicated option")
        if is_interactive:
            ui.wait()
        return

    ui.clear()
    ui.header()
    generator = Generator(client, ui)
    started_at = time.time()
    output_path = await generator.process(key, preferences)
    if output_path is not None:
        ui.clear()
        ui.header()
        ui.summary(output_path, generator.stats, time.time() - started_at)
    
    if is_interactive:
        ui.wait()

async def main() -> None:
    parser = _build_argument_parser()
    
    args_list = sys.argv[1:]
    if args_list and args_list[0] == "generate":
        args_list = args_list[1:]
        
    args = parser.parse_args(args_list)
    ui = ConsoleManager()

    async with NordClient() as client:
        if args.command == "get-key":
            await _run_get_key(ui, client, args.token or "")
        else:
            internal_groups = None
            if args.group:
                internal_groups = [ALIAS_TO_GROUP_ID[g] for g in args.group]
            prefs = UserPreferences(
                dns=args.dns,
                use_ip=args.ip,
                keepalive=args.keepalive,
                groups=internal_groups,
                exclude_dedicated=args.exclude_dedicated,
            )
            await _run_generate(ui, client, args.token or "", prefs)

def cli_entry_point() -> None:
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        pass

if __name__ == "__main__":
    cli_entry_point()