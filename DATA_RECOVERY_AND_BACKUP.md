# Data Recovery and Backup System Documentation

## Overview

This document describes the automated backup system and database safety improvements implemented to prevent data loss in the TodoMyDay application. The system includes automated backups, safer database migrations, and startup health checks.

---

## 🚨 CRITICAL: Data Recovery Steps

If you're experiencing data loss (0 rows in tables despite WAL file presence), follow these steps **immediately**:

### Recovery Commands

```bash
cd ~/mrbrain

# Step 1: Stop backend to release locks
sudo docker compose stop backend

# Step 2: Checkpoint the WAL file to merge data back into main database
sudo docker compose run --rm backend sqlite3 /data/todomyday.db "PRAGMA wal_checkpoint(TRUNCATE);"

# Step 3: Verify data is recovered
sudo docker compose run --rm backend sqlite3 /data/todomyday.db "SELECT COUNT(*) FROM memories;"
sudo docker compose run --rm backend sqlite3 /data/todomyday.db "SELECT COUNT(*) FROM users;"
sudo docker compose run --rm backend sqlite3 /data/todomyday.db "SELECT COUNT(*) FROM todos;"

# Step 4: If data is recovered, create immediate backup
sudo docker compose run --rm backend sqlite3 /data/todomyday.db ".backup /data/todomyday-recovered-$(date +%Y%m%d).db"

# Step 5: Copy backup to safe location
mkdir -p ~/backups
sudo docker compose cp backend:/data/todomyday-recovered-*.db ~/backups/

# Step 6: Start backend again
sudo docker compose start backend
```

### What This Does

- **WAL Checkpoint**: Merges uncommitted changes from the Write-Ahead Log (WAL) file back into the main database
- **TRUNCATE mode**: Ensures all WAL data is moved to the main database and the WAL file is reset
- **Verification**: Confirms data counts are correct
- **Backup**: Creates a snapshot of the recovered database

---

## Automated Backup System

### Components

1. **Internal Backup Script** - [backend/scripts/backup.sh](backend/scripts/backup.sh) - Runs inside Docker
2. **External Backup Script** - [backend/scripts/external-backup.sh](backend/scripts/external-backup.sh) - Runs on host machine
3. **Docker Backup Service** - Configured in [docker-compose.yml](docker-compose.yml#L51-L63)
4. **Startup Health Checks** - [backend/cmd/server/main.go](backend/cmd/server/main.go#L44-L80)
5. **Safe Migration Code** - [backend/internal/database/database.go](backend/internal/database/database.go#L297-L417)

### Backup Script Features

The backup script ([backend/scripts/backup.sh](backend/scripts/backup.sh)) includes:

- ✅ **WAL Checkpointing**: Ensures data consistency before backup
- ✅ **Integrity Verification**: Validates backup after creation
- ✅ **Detailed Logging**: All operations logged to `/data/backups/backup.log`
- ✅ **Automatic Cleanup**: Keeps only last 7 days of backups
- ✅ **Error Handling**: Proper exit codes and error messages
- ✅ **Size Tracking**: Reports backup sizes and counts

### Backup Schedule

**Docker Service Approach** (Default):
- Runs automatically every 2 hours (7200 seconds)
- Starts automatically with `docker-compose up`
- Portable across systems
- No manual configuration needed

**Cron Approach** (Alternative for specific times):
```bash
# Add to crontab (run: crontab -e)
0 2 * * * cd ~/mrbrain && sudo docker compose exec -T backend /app/scripts/backup.sh
```

Benefits of cron:
- Runs at specific time (2 AM daily)
- Lower resource overhead
- Email notifications on failure
- Better for predictable scheduling

### Backup Locations

#### Internal Backups (Inside Docker Volume)
- **Backup Directory**: `/data/backups/` (inside container) = `./data/backups/` (on host)
- **Log File**: `/data/backups/backup.log`
- **Backup Format**: `todomyday_YYYYMMDD_HHMMSS.db`
- **Retention**: 7 days (configurable in script)

#### External Backups (Outside Docker Volume)
- **Default Location**: `~/backups/todomyday/`
- **Format**: `YYYYMMDD_HHMMSS/` (timestamped directories)
- **Contents**: Main DB, WAL, automated backups, pre-migration backups, recovery backups
- **Retention**: 30 days (configurable in script)
- **Manifest**: Each backup includes `backup-manifest.txt` with file listing

---

## External Backup System (RECOMMENDED)

To protect against Docker volume corruption or deletion, set up external backups that copy data outside the Docker environment.

### Setup External Backups

The external backup script runs on your **host machine** (not inside Docker) and copies all database files to a safe location.

#### Option 1: Manual External Backup

```bash
# Run on production server
cd ~/mrbrain/backend/scripts

# Basic usage (backs up to ~/backups/todomyday/)
./external-backup.sh

# Custom backup location
./external-backup.sh /mnt/external-drive/todomyday-backups

# Or specify full path
./external-backup.sh /media/backup-drive/database-backups/todomyday
```

#### Option 2: Automated External Backups with Cron

Add to crontab for daily external backups at 3 AM:

```bash
# Edit crontab
crontab -e

# Add this line for daily backups at 3 AM
0 3 * * * cd ~/mrbrain/backend/scripts && ./external-backup.sh >> ~/backups/external-backup.log 2>&1
```

For weekly backups on Sundays at 2 AM:
```bash
0 2 * * 0 cd ~/mrbrain/backend/scripts && ./external-backup.sh >> ~/backups/external-backup.log 2>&1
```

#### Option 3: Cloud Storage Integration

The external backup script has commented sections for cloud uploads. Uncomment and configure:

**AWS S3:**
```bash
# Install AWS CLI first: sudo apt install awscli
# Configure: aws configure

# Edit external-backup.sh and uncomment:
aws s3 sync "${BACKUP_SUBDIR}" "s3://your-bucket/todomyday-backups/${BACKUP_TIMESTAMP}/"
```

**Google Cloud Storage:**
```bash
# Install gcloud CLI
# Authenticate: gcloud auth login

# Edit external-backup.sh and uncomment:
gsutil -m rsync -r "${BACKUP_SUBDIR}" "gs://your-bucket/todomyday-backups/${BACKUP_TIMESTAMP}/"
```

**Backblaze B2:**
```bash
# Install B2 CLI: pip install b2

# Edit external-backup.sh and uncomment:
b2 sync "${BACKUP_SUBDIR}" "b2://your-bucket/todomyday-backups/${BACKUP_TIMESTAMP}/"
```

**Remote Server (rsync):**
```bash
# Set up SSH key authentication first

# Edit external-backup.sh and uncomment:
rsync -avz "${BACKUP_SUBDIR}" user@backup-server:/backups/todomyday/${BACKUP_TIMESTAMP}/
```

### External Backup Features

✅ **Complete Snapshot**: Copies main DB, WAL file, automated backups, pre-migration backups, recovery backups
✅ **Timestamped**: Each backup in separate directory with timestamp
✅ **Manifest**: Includes detailed file listing and sizes
✅ **Retention Management**: Automatically removes backups older than 30 days
✅ **Compression Ready**: Uncomment tar.gz section for compressed backups
✅ **Cloud Integration**: Ready for S3, GCS, B2, or rsync
✅ **Colorized Logging**: Easy-to-read output with color-coded messages

### Viewing External Backups

```bash
# List all external backups
ls -lh ~/backups/todomyday/

# View specific backup contents
ls -lh ~/backups/todomyday/20260108_150530/

# Read backup manifest
cat ~/backups/todomyday/20260108_150530/backup-manifest.txt

# Check total external backup size
du -sh ~/backups/todomyday/
```

### Restoring from External Backup

```bash
# 1. Stop backend
cd ~/mrbrain
sudo docker compose stop backend

# 2. List available external backups
ls -lh ~/backups/todomyday/

# 3. Copy backup back to data directory (replace TIMESTAMP)
cp ~/backups/todomyday/YYYYMMDD_HHMMSS/todomyday.db ~/mrbrain/data/

# 4. Start backend
sudo docker compose start backend

# 5. Verify restoration
sudo docker compose exec backend sqlite3 /data/todomyday.db "SELECT COUNT(*) FROM memories;"
```

---

## Database Health Checks

### Startup Health Checks

On every backend startup, the following checks run automatically:

1. **WAL Checkpoint Status** - Attempts passive checkpoint and reports status
2. **Integrity Check** - Verifies database is not corrupted (`PRAGMA integrity_check`)
3. **Table Counts** - Logs row counts for users, todos, and memories
4. **Journal Mode** - Confirms database is in WAL mode

### Viewing Health Check Logs

```bash
# View backend startup logs
sudo docker compose logs backend --tail=50

# Look for these messages:
# - "Performing database health checks..."
# - "✅ Database integrity check passed"
# - "Database stats - Users: X, Todos: Y, Memories: Z"
# - "Database health checks complete"
```

---

## Safe Database Migration

The migration code has been rewritten to prevent data loss:

### Old Code Problems

- Used single `db.Exec()` with multi-statement SQL
- SQLite doesn't handle `BEGIN TRANSACTION`/`COMMIT` in single exec calls
- No backup created before migration
- No WAL checkpoint before migration
- Silent failures could lose data

### New Code Safety Features

1. ✅ **Pre-Migration Backup**: Creates backup using `VACUUM INTO` before any changes
2. ✅ **WAL Checkpoint**: Ensures data consistency with `PRAGMA wal_checkpoint(FULL)`
3. ✅ **Proper Transactions**: Uses `tx.Begin()` with step-by-step execution
4. ✅ **Error Handling**: Explicit `tx.Rollback()` on any error
5. ✅ **Data Verification**: Counts rows before/after copy to ensure no loss
6. ✅ **Detailed Logging**: Logs every step for debugging
7. ✅ **Panic Recovery**: Rollback on panic to prevent partial migrations

### Migration Backup Location

Pre-migration backups are stored as:
- **Location**: `/data/todomyday-pre-migration-YYYYMMDD-HHMMSS.db`
- **Format**: Timestamped with migration date/time
- **Retention**: Manual cleanup (not auto-deleted)

---

## Deployment Guide

### First-Time Setup

```bash
cd ~/mrbrain

# 1. Pull latest code with backup system
git pull

# 2. Rebuild images
sudo docker compose build

# 3. Restart services
sudo docker compose up -d

# 4. Verify backup service is running
sudo docker compose ps

# Expected output should show:
# - todomyday-backend (running)
# - todomyday-frontend (running)
# - todomyday-backup (running)
```

### Verifying Backup System

```bash
# Check backup service logs
sudo docker compose logs backup

# Manually trigger first backup (optional)
sudo docker compose exec backup /app/scripts/backup.sh

# View backup log
sudo docker compose exec backend cat /data/backups/backup.log

# List backups
sudo docker compose exec backend ls -lh /data/backups/*.db
```

---

## Monitoring and Maintenance

### Viewing Backup Logs

```bash
# View full backup log
sudo docker compose exec backend cat /data/backups/backup.log

# Tail backup log (live monitoring)
sudo docker compose exec backend tail -f /data/backups/backup.log

# View last 20 lines
sudo docker compose exec backend tail -n 20 /data/backups/backup.log
```

### Checking Backup Status

```bash
# List all backups with sizes
sudo docker compose exec backend ls -lh /data/backups/todomyday_*.db

# Count backups
sudo docker compose exec backend sh -c "ls -1 /data/backups/todomyday_*.db | wc -l"

# Check total backup size
sudo docker compose exec backend du -sh /data/backups
```

### Manual Backup

```bash
# Trigger backup immediately
sudo docker compose exec backend /app/scripts/backup.sh

# Or from backup service
sudo docker compose exec backup /app/scripts/backup.sh

# Create manual backup with custom name
sudo docker compose exec backend sqlite3 /data/todomyday.db ".backup /data/manual-backup-$(date +%Y%m%d).db"
```

### Restoring from Backup

```bash
# List available backups
sudo docker compose exec backend ls -lh /data/backups/todomyday_*.db

# Stop backend
sudo docker compose stop backend

# Restore from specific backup (replace YYYYMMDD_HHMMSS with actual timestamp)
sudo docker compose exec backend cp /data/backups/todomyday_YYYYMMDD_HHMMSS.db /data/todomyday.db

# Start backend
sudo docker compose start backend

# Verify restoration
sudo docker compose exec backend sqlite3 /data/todomyday.db "SELECT COUNT(*) FROM memories;"
```

---

## Troubleshooting

### Backup Service Not Running

```bash
# Check service status
sudo docker compose ps backup

# View service logs
sudo docker compose logs backup --tail=50

# Restart backup service
sudo docker compose restart backup
```

### No Backups Being Created

```bash
# Check if backup directory exists
sudo docker compose exec backend ls -la /data/backups/

# Check backup log for errors
sudo docker compose exec backend cat /data/backups/backup.log

# Manually run backup to see errors
sudo docker compose exec backup /app/scripts/backup.sh
```

### Database Integrity Issues

```bash
# Run integrity check
sudo docker compose exec backend sqlite3 /data/todomyday.db "PRAGMA integrity_check;"

# If corrupted, restore from latest backup
# (See "Restoring from Backup" section above)
```

### WAL File Growing Large

```bash
# Check WAL file size
sudo docker compose exec backend ls -lh /data/todomyday.db-wal

# Checkpoint WAL manually
sudo docker compose exec backend sqlite3 /data/todomyday.db "PRAGMA wal_checkpoint(FULL);"

# Health checks on startup also checkpoint WAL automatically
sudo docker compose restart backend
```

---

## Configuration Options

### Adjusting Backup Frequency

Edit [docker-compose.yml](docker-compose.yml#L60):

```yaml
# Current: Every 2 hours (7200 seconds)
command: sh -c "while true; do sleep 7200 && /app/scripts/backup.sh; done"

# Daily (86400 seconds)
command: sh -c "while true; do sleep 86400 && /app/scripts/backup.sh; done"

# Every 6 hours (21600 seconds)
command: sh -c "while true; do sleep 21600 && /app/scripts/backup.sh; done"
```

Then restart:
```bash
sudo docker compose up -d backup
```

### Adjusting Retention Period

Edit [backend/scripts/backup.sh](backend/scripts/backup.sh#L119):

```bash
# Current: Keep 7 days
find ${BACKUP_DIR} -name "todomyday_*.db" -mtime +7 -type f -delete

# Keep 14 days
find ${BACKUP_DIR} -name "todomyday_*.db" -mtime +14 -type f -delete

# Keep 30 days
find ${BACKUP_DIR} -name "todomyday_*.db" -mtime +30 -type f -delete
```

Rebuild and restart:
```bash
sudo docker compose build backup
sudo docker compose restart backup
```

---

## File Reference

### Created Files
- [backend/scripts/backup.sh](backend/scripts/backup.sh) - Automated backup script

### Modified Files
- [backend/internal/database/database.go](backend/internal/database/database.go#L297-L417) - Safe migration code
- [backend/cmd/server/main.go](backend/cmd/server/main.go#L44-L80) - Startup health checks
- [docker-compose.yml](docker-compose.yml#L51-L63) - Backup service configuration

### Related Files
- [backend/Dockerfile](backend/Dockerfile#L39-L41) - Script copying configuration
- [test-prod-auth.sh](test-prod-auth.sh) - Production authentication testing

---

## Best Practices

### Regular Monitoring

1. Check backup logs weekly: `sudo docker compose exec backend cat /data/backups/backup.log`
2. Verify backup sizes are reasonable (should grow with data)
3. Test restore process monthly (restore to test environment)
4. Monitor disk space: `df -h ./data`

### Before Major Changes

1. Create manual backup: `sudo docker compose exec backend /app/scripts/backup.sh`
2. Copy backup to external storage
3. Note the backup filename for rollback if needed

### Production Deployment

1. Always recover existing data first (see "Data Recovery Steps")
2. Test in local environment before deploying to production
3. Create backup immediately before deployment
4. Monitor logs after deployment for any issues

---

## Additional Resources

- [SUPABASE_SETUP.md](SUPABASE_SETUP.md) - Supabase authentication setup
- [CLAUDE.md](CLAUDE.md) - Project overview and development guide
- SQLite WAL Mode: https://www.sqlite.org/wal.html
- SQLite Backup API: https://www.sqlite.org/backup.html

---

## Support

If you encounter issues not covered in this documentation:

1. Check backup logs: `/data/backups/backup.log`
2. Check backend logs: `sudo docker compose logs backend --tail=100`
3. Review startup health checks in logs
4. Ensure sufficient disk space: `df -h`

For data recovery emergencies, prioritize running the WAL checkpoint commands at the top of this document.
