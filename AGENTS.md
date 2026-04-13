# DBStat Project - Agent Guidelines

## Project Overview
DBStat is a desktop application for monitoring and managing database services (PostgreSQL, MySQL/MariaDB, Redis) on Linux systems using systemd. It provides a GUI interface built with the Fyne toolkit in Go.

## Development Setup

### Prerequisites
- Go 1.24.5 or later
- Fyne GUI toolkit (v2.7.2)
- Linux system with systemd
- Optional: PostgreSQL, MySQL/MariaDB, Redis services for testing

### Getting Started
1. Clone the repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the application:
   ```bash
   go build -o dbstat main.go
   ```
4. Run the application:
   ```bash
   ./dbstat
   ```

## Code Structure

### Key Files
- `main.go`: Main application logic and GUI
- `go.mod`: Go module definition and dependencies
- `assets/`: SVG icons for database services
- `build/`: Build artifacts and packaging files
- `build.sh`: Script for building the application
- `install.sh`: Script for installing the application

### Fyne GUI Components
- Uses Fyne v2 for cross-platform desktop GUI
- Embedded SVG assets via `//go:embed` directives
- Dynamic UI updates based on service status
- System tray integration (via indirect dependency)

## System Integration
- Monitors systemd services for PostgreSQL, MySQL/MariaDB, Redis
- Uses `systemctl` commands to:
  - Check service status (`systemctl is-active`)
  - Get service PID (`systemctl show --property=MainPID`)
  - Start/stop/restart services (with pkexec for privilege escalation)
- Port detection via `netstat` parsing
- Falls back to default ports when services are running but port detection fails

## Building and Packaging

### Development Build
```bash
go build -o dbstat main.go
```

### Release Build (using build.sh)
```bash
./build.sh
```
Creates:
- Binary executable in `build/dbstat`
- Desktop entry file (`build/dbstat.desktop`)
- Icon files (`build/icon.png`, `build/icon.svg`)

### Installation
```bash
sudo ./install.sh
```
Installs to `/usr/local/bin/dbstat` and creates application menu entry.

## Testing

### Manual Testing
1. Ensure database services are installed (PostgreSQL, MySQL, Redis)
2. Start some services: `sudo systemctl start postgresql mysql redis`
3. Run dbstat: `./dbstat`
4. Verify service status, ports, and control buttons work correctly
5. Test stopping/starting services through the UI

### Service Detection Logic
The application detects services by:
1. Looking for systemd units with prefixes:
   - PostgreSQL: `postgresql`, `postgres`
   - MySQL/MariaDB: `mysql`, `mariadb`
   - Redis: `redis`, `redis-server`
2. For each found unit, it retrieves:
   - Status (active/inactive/failed)
   - Main PID
   - Listening port (via netstat or service name fallback)
   - Default ports as last resort

## Contributing Guidelines

### Code Style
- Follow standard Go formatting (gofmt)
- Keep functions focused and readable
- Comment complex logic sections
- Maintain consistent error handling patterns

### Making Changes
1. Create a feature branch
2. Make your changes
3. Ensure the application builds: `go build`
4. Test manually with actual services
5. Update documentation if needed
6. Submit pull request

### Dependencies
- Direct dependency: `fyne.io/fyne/v2 v2.7.2`
- Indirect dependencies managed by Go modules
- To update dependencies: `go get -u ./...` then `go mod tidy`

## Troubleshooting

### Common Issues
1. **No services detected**: 
   - Ensure database services are installed and registered with systemd
   - Check service names match expected patterns
   - Run `systemctl list-unit-files | grep -E '(postgres|mysql|redis)'`

2. **Port detection fails**:
   - Application falls back to default ports (5432, 3306, 6379)
   - Verify netstat access permissions
   - Check if services are actually listening on expected ports

3. **Permission errors starting/stopping services**:
   - Application uses pkexec for privilege escalation
   - Ensure pkexec is installed and configured properly
   - Test manual service control: `sudo systemctl restart <service>`

4. **GUI display issues**:
   - Fyne requires proper graphics drivers
   - Test with `fyne-demo` if available
   - Ensure required GTK libraries are installed

## Notes for AI Agents
- When modifying service detection logic, update both prefixes and process patterns
- GUI changes should maintain responsiveness and proper layout
- Always test with actual running services when possible
- Keep error handling robust for cases where services aren't running
- Consider cross-platform compatibility if expanding beyond Linux/systemd