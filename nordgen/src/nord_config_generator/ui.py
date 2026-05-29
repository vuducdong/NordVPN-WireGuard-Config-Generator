from contextlib import contextmanager
from typing import Iterator

import pwinput
from rich.console import Console
from rich.panel import Panel
from rich.progress import BarColumn, Progress, SpinnerColumn, TextColumn
from rich.prompt import Confirm, IntPrompt, Prompt
from rich.table import Table
from rich.theme import Theme

from .models import GenerationStats, UserPreferences

_THEME = Theme({
    "info": "bright_cyan",
    "success": "bold bright_green",
    "warning": "bright_yellow",
    "error": "bold bright_red",
    "title": "bold bright_white",
})


class ConsoleManager:
    def __init__(self) -> None:
        self.console = Console(theme=_THEME)

    def clear(self) -> None:
        self.console.clear()

    def header(self) -> None:
        self.console.print(Panel(
            "[title]NordVPN Configuration Generator[/title]",
            expand=False,
            border_style="bright_cyan",
            padding=(0, 2),
        ))

    def prompt_secret(self, message: str) -> str:
        return pwinput.pwinput(prompt=f"\033[96m{message}: \033[0m", mask="*").strip()

    def prompt_preferences(self, defaults: UserPreferences) -> UserPreferences:
        self.console.print("[info]Configuration Options (Enter for default)[/info]")
        dns = Prompt.ask("DNS IP", console=self.console, default=defaults.dns).strip()
        use_ip = Confirm.ask("Use IP for endpoints?", console=self.console, default=defaults.use_ip)
        keepalive = IntPrompt.ask(
            "PersistentKeepalive", console=self.console, default=defaults.keepalive
        )
        exclude_dedicated = Confirm.ask(
            "Exclude dedicated IP servers?", console=self.console, default=defaults.exclude_dedicated
        )
        return UserPreferences(
            dns=dns,
            use_ip=use_ip,
            keepalive=keepalive,
            groups=defaults.groups,
            exclude_dedicated=exclude_dedicated,
        )

    @contextmanager
    def status(self, message: str) -> Iterator[None]:
        with self.console.status(f"[bright_cyan]{message}[/bright_cyan]"):
            yield

    @contextmanager
    def progress(self) -> Iterator[Progress]:
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            BarColumn(),
            TextColumn("{task.completed}/{task.total}"),
            console=self.console,
            transient=False,
        ) as progress_instance:
            yield progress_instance

    def success(self, message: str) -> None:
        self.console.print(f"[success]{message}[/success]")

    def fail(self, message: str) -> None:
        self.console.print(f"[error]{message}[/error]")

    def error(self, message: str) -> None:
        self.console.print(f"[error]{message}[/error]")

    def show_key(self, key: str) -> None:
        self.console.print(Panel(
            f"[bright_green]{key}[/bright_green]",
            title="NordLynx Private Key",
            border_style="bright_green",
            expand=False,
        ))

    def summary(self, output_path: str, stats: GenerationStats, duration_seconds: float) -> None:
        grid = Table.grid(padding=(0, 2))
        grid.add_column(style="bright_cyan")
        grid.add_column()
        grid.add_row("Output Directory:", output_path)
        grid.add_row("Standard Configs:", str(stats.total))
        grid.add_row("Optimized Configs:", str(stats.best))
        grid.add_row("Duration:", f"{duration_seconds:.2f}s")
        self.console.print(Panel(grid, title="Complete", border_style="bright_green", expand=False))

    def wait(self) -> None:
        self.console.print()
        try:
            self.console.input("[info]Press Enter to exit... [/info]")
        except (KeyboardInterrupt, EOFError):
            pass