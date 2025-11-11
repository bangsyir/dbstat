# DB Start

DB Start is a simple Fyne GUI application designed to manage the status of common database engines (PostgreSQL, MySQL/MariaDB, Redis) on Linux systems using `systemctl`. It provides a graphical interface to start, stop, or restart these services and displays their current status and listening port.

## Features

- **Discover Database Services**: Automatically identifies installed PostgreSQL, MySQL/MariaDB, and Redis services.
- **Systemctl Integration**: Uses `systemctl` to control services (start, stop, restart).
- **Real-time Status Display**: Shows the active status (running, inactive, failed) and listening port of each service.
- **Fyne GUI**: A cross-platform GUI built with the Fyne toolkit.
- **`pkexec` for Permissions**: Leverages `pkexec` to securely execute `systemctl` commands that require root privileges.

## Supported Database Engines

- PostgreSQL
- MySQL / MariaDB
- Redis

## Prerequisites

- **Linux Operating System**: This application is designed for Linux environments.
- **`systemd`**: Services are managed via `systemd`.
- **`systemctl`**: Command-line interface for `systemd`.
- **`pkexec`**: For privilege escalation to manage services.
- **`ss`**: For checking listening ports.
- **Go**: To build and run the application.
- **Fyne Dependencies**: Required libraries for Fyne applications.

  On Debian/Ubuntu-based systems, you might need:

  ```/dev/null/install_deps.sh
  sudo apt-get update
  sudo apt-get install build-essential libgl1-mesa-dev xorg-dev
  ```

  For other distributions, refer to the [Fyne documentation](https://developer.fyne.io/started/dependencies).

## Installation and Running

1.  **Clone the repository**:

    ```/dev/null/clone.sh
    git clone https://github.com/bangsyir/dbstat.git
    cd dbstat
    ```

2.  **Build the application**:

    This script compiles the application and creates an executable binary in the `build/` directory, along with a desktop entry and icon.

    ```dbstat/build.sh
    ./build.sh
    ```

3.  **Install the application (optional)**:

    This script installs the built application, its desktop entry, and icon into your user's local directories (`~/.local/bin`, `~/.local/share/applications`, `~/.local/share/icons`). This makes it available in your system's application menu.

    ```dbstat/install.sh
    ./install.sh
    ```

4.  **Run the application**:
    - If you ran `./install.sh`, you can find "DB Stat Manager" in your desktop environment's application menu.
    - Alternatively, you can run the application directly from the build directory:

      ```/dev/null/run_from_build.sh
      ./build/dbstat
      ```

## Usage

Upon running the application, a window will appear displaying a list of detected database services. Each row will show:

- The database
