# DBStat

DBStat is a desktop application for monitoring and managing database services (PostgreSQL, MySQL/MariaDB, Redis) on Linux systems using systemd. It provides a GUI to view service status, ports, and control (start/stop/restart) these services.

## Features

- **Service Discovery**: Automatically detects installed PostgreSQL, MySQL/MariaDB, and Redis services via systemd.
- **Status Monitoring**: Displays real-time status (active/inactive/failed), PID, and listening port for each service.
- **Service Control**: Start, stop, or restart services directly from the GUI using `pkexec` for privilege escalation.
- **Fyne GUI**: Cross-platform desktop GUI built with the Fyne toolkit.
- **Activity Log**: Shows real-time log entries for service operations.

## Supported Database Engines

- PostgreSQL
- MySQL / MariaDB
- Redis

## Prerequisites

- **Linux**: This application runs on Linux with systemd.
- **Go 1.24+**: To build the application.
- **Fyne Dependencies**: See [Fyne Quick Start](https://docs.fyne.io/started/quick/) for installation instructions.

## Installation and Running

1.  **Clone the repository**:

    ```bash
    git clone https://github.com/bangsyir/dbstat.git
    cd dbstat
    ```

2.  **Install dependencies**:

    ```bash
    go mod tidy
    ```

3.  **Build the application**:

    ```bash
    go build -o dbstat main.go
    ```

    Or use the build script for release artifacts:

    ```bash
    ./build.sh
    ```

4.  **Install the application (optional)**:

    ```bash
    sudo ./install.sh
    ```

    This installs to `/usr/local/bin` and creates a desktop entry.

5.  **Run the application**:

    ```bash
    ./dbstat
    ```

    Or find "DB Stat Manager" in your desktop environment's application menu if installed.

## Usage

Upon launching, the application displays detected database services in a list. Each row shows:

- Service icon and name
- Status indicator (active/inactive/failed)
- Listening port
- Control button to start/stop/restart

The activity panel at the bottom displays real-time log entries for all operations.

The application auto-refreshes service status every few seconds. To control services, click the action button and enter your password when prompted via the polkit dialog.
