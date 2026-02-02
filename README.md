# Ticketbot

A notification service that sends ConnectWise ticket updates to Webex. When tickets are created or updated in ConnectWise, Ticketbot delivers real-time notifications to designated Webex rooms or individual users.

## Components

- **Server**: Background service that monitors ConnectWise for ticket changes and sends notifications via Webex
- **CLI/TUI**: Administrative tool for managing notification rules, user forwarding, API keys, and data synchronization

## Installation

### CLI Installation

```bash
go install github.com/thecoretg/ticketbot/cmd/tbot-admin@latest
```

### Server Deployment

The server is deployed to AWS Lightsail using the provided Makefile:

```bash
make deploy-lightsail
```

This command builds a Docker image and pushes it to the Lightsail container service in the us-west-2 region.

## How It Works

### Notification Flow

1. **Webhook Reception**: ConnectWise sends a webhook when a ticket is created or updated
2. **Ticket Processing**: Server fetches full ticket details from ConnectWise API and stores them in PostgreSQL
3. **Rule Matching**: Server looks up notification rules for the ticket's board
4. **Recipient Resolution**: Determines which Webex rooms or users should receive notifications (behavior differs for new vs. updated tickets)
5. **Forward Processing**: Applies user forwarding rules to redirect notifications
6. **Message Delivery**: Sends formatted messages to Webex recipients
7. **Notification Tracking**: Records sent notifications to prevent duplicates

### New vs. Updated Ticket Notifications

The system handles new tickets and ticket updates differently:

**New Tickets:**
- Notifications are sent to all Webex recipients defined in rules for the ticket's board
- Notifications are also sent to ConnectWise members assigned to the ticket (resources)
- Establishes initial awareness of the ticket across configured channels

**Updated Tickets:**
- Notifications are sent ONLY to ConnectWise members assigned to the ticket (resources)
- Rule-based recipients (rooms) do NOT receive update notifications
- The member who created the latest note is excluded from receiving their own notification
- Duplicate prevention: If a notification was already sent for a specific note ID, no duplicate is sent
- Focuses updates on individuals actively working on the ticket rather than broadcasting to all rooms

### Rules

Rules define which Webex recipients receive notifications for tickets on specific ConnectWise boards. Each rule maps a board to a recipient (either a Webex room or individual user).

- **Board-to-Recipient Mapping**: One rule per board-recipient pair
- **Enable/Disable Toggle**: Rules can be temporarily disabled without deletion
- **Active Filtering**: Only enabled rules are evaluated during notification processing

Example: A rule mapping the "IT Support" board to the "Help Desk" Webex room ensures all tickets on that board generate notifications in that room.

### Forwards

Forwards redirect notifications from one Webex recipient to another, typically used when users are unavailable. Forwards support:

- **Source and Destination**: Redirect from one recipient to another recipient
- **Copy Retention**: Option for source recipient to keep a copy of the notification
- **Date Ranges**: Automatic activation and deactivation based on start/end dates
- **Chain Prevention**: System detects and prevents circular forwarding chains
- **Conditional Application**: If the destination recipient would already receive the notification through rules, the forward is skipped to avoid duplicates

Example: User A is on vacation from March 1-15. A forward is configured from User A to User B with those dates. During this period, any notifications that would go to User A are sent to User B instead. If "keep copy" is enabled, User A still receives the notifications.

## Sync Workflows

Ticketbot requires periodic synchronization to keep local data in sync with ConnectWise and Webex. Three sync operations are available:

### Board Sync
Synchronizes ConnectWise boards from the PSA system into the local database. This operation:
- Fetches all boards from ConnectWise API
- Upserts board records into PostgreSQL
- Syncs board statuses for each board
- Soft-deletes boards that no longer exist in ConnectWise

### Recipient Sync
Synchronizes Webex rooms and people into the local database. This operation:
- Fetches all Webex rooms from the Webex API
- Fetches Webex people details for each room member
- Upserts recipient records into PostgreSQL
- Enables notification targeting to rooms and individuals

### Ticket Sync
Synchronizes ConnectWise tickets into the local database. This operation:
- Fetches open tickets from specified boards (or all boards if none specified)
- Processes each ticket including notes, resources, and metadata
- Stores ticket data in PostgreSQL
- This is the longest-running sync operation (5-30 minutes depending on ticket volume)

### Running Syncs

Syncs can be triggered via CLI or TUI:

```bash
# Sync everything
tbot-admin sync --all

# Sync specific resources
tbot-admin sync --boards --recipients
tbot-admin sync --tickets --sync-boards 1,2,3

# Check sync status
tbot-admin sync status
```

### Automated Sync Recommendation

It is recommended to schedule sync operations using a cron server or similar job scheduler. Example cron configuration:

```cron
# Sync boards and recipients daily at 2 AM
0 2 * * * tbot-admin sync --boards --recipients

# Sync tickets weekly on Sunday at 3 AM
0 3 * * 0 tbot-admin sync --tickets --all
```

This ensures the local database stays current with external systems without manual intervention.

## Authentication

### User Creation

API users are required to authenticate with the server. Users can be created via CLI or TUI:

```bash
tbot-admin create user --name "Service Account" --email "service@example.com"
```

Each user can have multiple API keys. Users are primarily used for access control and audit logging.

### API Key Creation

API keys authenticate requests to the server. Keys are tied to specific users:

```bash
# Create a key for user ID 1
tbot-admin create key --user-id 1
```

The CLI returns the full API key on creation. This is the only time the key is shown in plaintext. The server stores a bcrypt hash of the key for verification.

API keys must be included in the `Authorization` header for all authenticated requests:

```
Authorization: Bearer <api-key>
```

## API Endpoints

All endpoints except `/healthcheck` and `/hooks/cw/tickets` require API key authentication.

### Health & Auth
- `GET /healthcheck` - Unauthenticated health check
- `GET /authtest` - Authenticated health check

### Sync
- `POST /sync` - Trigger sync operation (body: SyncPayload)
- `GET /sync/status` - Get current sync status

### Users & Keys
- `GET /users` - List all users
- `GET /users/me` - Get current authenticated user
- `GET /users/:id` - Get user by ID
- `POST /users` - Create new user
- `DELETE /users/:id` - Delete user
- `GET /users/keys` - List all API keys
- `GET /users/keys/:id` - Get API key by ID
- `POST /users/keys` - Create new API key
- `DELETE /users/keys/:id` - Delete API key

### ConnectWise Data
- `GET /cw/boards` - List synced boards
- `GET /cw/boards/:id` - Get board by ID
- `GET /cw/members` - List ConnectWise members

### Webex Data
- `GET /webex/rooms` - List synced Webex recipients
- `GET /webex/rooms/:id` - Get recipient by ID

### Notification Rules
- `GET /notifiers/rules` - List all rules
- `GET /notifiers/rules/:id` - Get rule by ID
- `POST /notifiers/rules` - Create new rule
- `DELETE /notifiers/rules/:id` - Delete rule

### Forwards
- `GET /notifiers/forwards` - List all forwards
- `GET /notifiers/forwards/:id` - Get forward by ID
- `POST /notifiers/forwards` - Create new forward
- `DELETE /notifiers/forwards/:id` - Delete forward

### Configuration
- `GET /config` - Get server configuration
- `PUT /config` - Update server configuration

### Webhooks
- `POST /hooks/cw/tickets` - ConnectWise ticket webhook (requires CW signature)

## Configuration

The server requires the following environment variables:

- ConnectWise API credentials
- Webex Bot token
- PostgreSQL database connection string
- Server port and other service configurations

## CLI Usage

The `tbot-admin` CLI provides commands for managing the ticketbot service:

### Basic Commands

```bash
# Check authentication
tbot-admin authcheck

# List resources
tbot-admin list boards
tbot-admin list rules
tbot-admin list recipients
tbot-admin list users
tbot-admin list keys

# Create resources
tbot-admin create rule --board-id <id> --recipient-id <id>
tbot-admin create user --name <name> --email <email>
tbot-admin create key --user-id <id>

# Delete resources
tbot-admin delete rule <id>
tbot-admin delete user <id>
tbot-admin delete key <id>

# Update resources
tbot-admin update rule <id> --enabled=true|false

# Sync data from external sources
tbot-admin sync --boards --recipients --tickets

# Test connectivity
tbot-admin ping
```

### TUI Mode

Launch the interactive terminal UI by running the command without any subcommands:

```bash
tbot-admin
```

The TUI provides a visual interface for managing:

- **Rules**: Map ConnectWise boards to Webex recipients for notifications
- **Forwards**: Configure user-to-user notification forwarding
- **Users**: Manage API users
- **Keys**: Generate and manage API keys
- **Sync**: Synchronize boards, recipients, and tickets from ConnectWise and Webex

#### TUI Navigation

- `CTRL+R`: Switch to Rules view
- `CTRL+F`: Switch to Forwards view
- `CTRL+U`: Switch to Users view
- `CTRL+A`: Switch to API Keys view
- `CTRL+S`: Switch to Sync view
- `N`: Create new item
- `X`: Delete selected item
- `CTRL+C`: Quit

## Development

### Build

```bash
# Build server
make build-server

# Build CLI
make build-cli
```

### Local Development

```bash
# Run server locally
make runserver

# Run TUI locally
make tui

# Start test database
make test-db-up
```

### View Logs

```bash
# View Lightsail container logs
make lightsail-logs
```
