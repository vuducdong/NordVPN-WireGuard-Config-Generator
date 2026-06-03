import asyncio
import os
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
from datetime import datetime

from rich.progress import Progress, TaskID

from .client import NordClient
from .models import GenerationStats, Server, UserPreferences
from .server_parser import parse_servers
from .ui import ConsoleManager

_PATH_SANITIZE_TABLE: dict[int, int | None] = {ord(c): None for c in '<>:"/\\|?*\0'}
_PATH_SANITIZE_TABLE[ord(" ")] = ord("_")

_FILENAME_MAX_LENGTH = 15


@dataclass(slots=True, frozen=True)
class _ConfigWriteJob:
    absolute_path: str
    content: str


class Generator:
    def __init__(self, client: NordClient, ui: ConsoleManager) -> None:
        self.client = client
        self.ui = ui
        self.stats = GenerationStats()
        self.output_directory: str = ""

    async def process(
        self, private_key: str, preferences: UserPreferences
    ) -> str | None:
        self.output_directory = f"nordvpn_configs_{datetime.now().strftime('%Y%m%d_%H%M%S')}"

        with self.ui.status("Fetching data..."):
            (latitude, longitude), raw_servers = await asyncio.gather(
                self.client.get_geo(),
                self.client.get_servers(),
            )

        if not raw_servers:
            self.ui.fail("Failed to fetch server data")
            return None
        self.ui.success("Data fetched")

        with self.ui.status("Processing dataset..."):
            required_groups = set(preferences.groups) if preferences.groups else None

            all_parsed = parse_servers(
                raw_servers,
                latitude,
                longitude,
                required_groups=required_groups,
                exclude_dedicated=preferences.exclude_dedicated,
            )

            unique_servers = list({s.name: s for s in all_parsed}.values())
            
            if not unique_servers:
                self.ui.fail("No servers found matching the specified filters")
                return None
                
            unique_servers.sort(key=lambda s: (s.load, s.distance))

            self.stats.total = len(unique_servers)

            best_map: dict[tuple[str, str, str], Server] = {}
            for s in unique_servers:
                key = (s.combo, s.country, s.city)
                if key not in best_map:
                    best_map[key] = s
            self.stats.best = len(best_map)

            standard_jobs = self._build_jobs(
                unique_servers, "configs", private_key, preferences
            )
            best_jobs = self._build_jobs(
                list(best_map.values()), "best_configs", private_key, preferences
            )

        try:
            all_jobs = standard_jobs + best_jobs
            await asyncio.to_thread(
                self._materialize_directories, all_jobs
            )

            with self.ui.progress() as progress:
                task = progress.add_task("Writing all configs", total=len(all_jobs))
                await asyncio.to_thread(
                    self._write_jobs_parallel, all_jobs, progress, task
                )
        except OSError as err:
            self.ui.fail(f"Filesystem error: {err}")
            return None

        return self.output_directory

    def _build_jobs(
        self,
        servers: list[Server],
        subdirectory: str,
        private_key: str,
        preferences: UserPreferences,
    ) -> list[_ConfigWriteJob]:
        jobs: list[_ConfigWriteJob] = []
        counts: dict[str, int] = {}
        base_dir = self.output_directory

        interface_block = f"[Interface]\nPrivateKey = {private_key}\nAddress = 10.5.0.2/16\nDNS = {preferences.dns}\n\n[Peer]\n"
        keepalive_block = f"\nPersistentKeepalive = {preferences.keepalive}"

        for server in servers:
            country_seg = server.country.lower().translate(_PATH_SANITIZE_TABLE)
            city_seg = server.city.lower().translate(_PATH_SANITIZE_TABLE)
            fname_root = server.name.lower().translate(_PATH_SANITIZE_TABLE)[:_FILENAME_MAX_LENGTH]
            
            if not fname_root:
                fname_root = "unknown"

            directory = os.path.join(
                base_dir, subdirectory, server.combo, country_seg, city_seg
            )
            candidate = os.path.join(directory, f"{fname_root}.conf")

            count = counts.get(candidate, 0)
            counts[candidate] = count + 1
            final_path = candidate if count == 0 else os.path.join(directory, f"{fname_root}_{count}.conf")

            endpoint = server.station if preferences.use_ip else server.hostname
            content = f"{interface_block}PublicKey = {server.public_key}\nAllowedIPs = 0.0.0.0/0, ::/0\nEndpoint = {endpoint}:51820{keepalive_block}"
            jobs.append(_ConfigWriteJob(absolute_path=final_path, content=content))
        return jobs

    @staticmethod
    def _materialize_directories(jobs: list[_ConfigWriteJob]) -> None:
        unique_dirs = {os.path.dirname(job.absolute_path) for job in jobs}
        for d in unique_dirs:
            os.makedirs(d, exist_ok=True)

    @staticmethod
    def _write_jobs_chunk(jobs: list[_ConfigWriteJob], progress: Progress, task_id: TaskID) -> None:
        for job in jobs:
            with open(job.absolute_path, "w", encoding="utf-8") as f:
                f.write(job.content)
        progress.advance(task_id, len(jobs))

    @classmethod
    def _write_jobs_parallel(
        cls, jobs: list[_ConfigWriteJob], progress: Progress, task_id: TaskID
    ) -> None:
        chunk_size = 50
        cpu_count = os.cpu_count() or 1
        max_workers = min(64, max(4, cpu_count * 4))
        
        chunks = [jobs[i : i + chunk_size] for i in range(0, len(jobs), chunk_size)]
        
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = [
                executor.submit(cls._write_jobs_chunk, chunk, progress, task_id)
                for chunk in chunks
            ]
            for future in futures:
                future.result()