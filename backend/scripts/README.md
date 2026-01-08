# Backup Scripts

This directory contains scripts for database backup and management.

## Scripts Overview

### 1. backup.sh (Internal Backups)
**Location**: Runs inside Docker container
**Purpose**: Automated frequent backups within Docker volume
**Frequency**: Every 2 hours (configurable in docker-compose.yml)
**Retention**: 7 days

**Features**:
- WAL checkpointing before backup
- Integrity verification after backup
- Detailed logging to `/data/backups/backup.log`
- Automatic cleanup of old backups

**Usage**:
```bash
# Automatically runs via Docker Compose backup service
# Manual trigger:
sudo docker compose exec backup /app/scripts/backup.sh
```

---

### 2. external-backup.sh (External Backups)
**Location**: Runs on host machine (outside Docker)
**Purpose**: Protection against Docker volume corruption/deletion
**Frequency**: Recommended daily via cron
**Retention**: 30 days

**Features**:
- Complete snapshot (main DB, WAL, all backups)
- Timestamped directories
- Backup manifest with file listing
- Optional compression
- Cloud storage integration ready
- Colorized output

**Usage**:
```bash
# Basic (backs up to ~/backups/todomyday/)
./external-backup.sh

# Custom location
./external-backup.sh /mnt/external-drive/backups

# Add to crontab for automation (daily at 3 AM)
0 3 * * * cd ~/mrbrain/backend/scripts && ./external-backup.sh >> ~/backups/external-backup.log 2>&1
```

---

### 3. check_users.go
**Purpose**: View users in database
**Usage**:
```bash
sudo docker compose exec backend /app/check-users
```

---

### 4. check_memories.go
**Purpose**: View memories in database with statistics
**Usage**:
```bash
sudo docker compose exec backend /app/check-memories
```

---

### 5. migrate_users_to_supabase.go
**Purpose**: Migrate local users to Supabase
**Usage**:
```bash
sudo docker compose exec backend /app/migrate-users
```

---

## Backup Strategy

### Three-Tier Approach

1. **Internal Backups** (backup.sh)
   - Fast recovery
   - Frequent (2 hours)
   - Short retention (7 days)
   - Inside Docker volume

2. **External Backups** (external-backup.sh)
   - Volume protection
   - Daily
   - Longer retention (30 days)
   - Outside Docker on host filesystem

3. **Cloud Backups** (optional)
   - Disaster recovery
   - Off-site storage
   - Configurable retention
   - Add to external-backup.sh

### Recommended Setup

```bash
# 1. Internal backups run automatically (Docker Compose)
# Already configured - no action needed

# 2. Setup external backups
mkdir -p ~/backups/todomyday
chmod +x ~/mrbrain/backend/scripts/external-backup.sh

# 3. Add cron job for daily external backups
crontab -e
# Add: 0 3 * * * cd ~/mrbrain/backend/scripts && ./external-backup.sh >> ~/backups/external-backup.log 2>&1

# 4. (Optional) Configure cloud storage
# Edit external-backup.sh and uncomment your preferred cloud provider
```

---

## Monitoring

### Check Internal Backups
```bash
# View backup log
sudo docker compose exec backend cat /data/backups/backup.log

# List backups
sudo docker compose exec backend ls -lh /data/backups/

# Check backup service status
sudo docker compose ps backup
```

### Check External Backups
```bash
# View backup log
cat ~/backups/external-backup.log

# List backups
ls -lh ~/backups/todomyday/

# Check disk space
df -h ~/backups
```

---

## Recovery

### From Internal Backup
```bash
# Stop backend
sudo docker compose stop backend

# List backups
sudo docker compose exec backend ls -lh /data/backups/

# Restore (replace TIMESTAMP)
sudo docker compose exec backend cp /data/backups/todomyday_TIMESTAMP.db /data/todomyday.db

# Start backend
sudo docker compose start backend
```

### From External Backup
```bash
# Stop backend
sudo docker compose stop backend

# List backups
ls -lh ~/backups/todomyday/

# Restore (replace TIMESTAMP)
cp ~/backups/todomyday/TIMESTAMP/todomyday.db ~/mrbrain/data/

# Start backend
sudo docker compose start backend
```

---

## Configuration

### Adjust Internal Backup Frequency
Edit `docker-compose.yml`:
```yaml
# Change sleep time (in seconds)
command: sh -c "while true; do sleep 7200 && /app/scripts/backup.sh; done"
# 3600 = 1 hour
# 7200 = 2 hours (current)
# 21600 = 6 hours
# 86400 = 24 hours
```

### Adjust Internal Backup Retention
Edit `backup.sh` line 119:
```bash
# Change days to keep
find ${BACKUP_DIR} -name "todomyday_*.db" -mtime +7 -type f -delete
# +3 = 3 days
# +7 = 7 days (current)
# +14 = 14 days
```

### Adjust External Backup Retention
Edit `external-backup.sh` line 16:
```bash
# Change retention days
RETENTION_DAYS=30  # Current: 30 days
```

### Enable Compression
Edit `external-backup.sh` around line 107, uncomment:
```bash
log_info "Compressing backup..."
tar -czf "${EXTERNAL_BACKUP_DIR}/backup_${BACKUP_TIMESTAMP}.tar.gz" -C "${EXTERNAL_BACKUP_DIR}" "${BACKUP_TIMESTAMP}"
rm -rf "${BACKUP_SUBDIR}"
```

---

## Cloud Storage Setup

### AWS S3
```bash
# Install and configure
sudo apt install awscli
aws configure

# Edit external-backup.sh, uncomment:
aws s3 sync "${BACKUP_SUBDIR}" "s3://your-bucket/todomyday-backups/${BACKUP_TIMESTAMP}/"
```

### Google Cloud Storage
```bash
# Install and configure
# Follow: https://cloud.google.com/sdk/docs/install
gcloud auth login

# Edit external-backup.sh, uncomment:
gsutil -m rsync -r "${BACKUP_SUBDIR}" "gs://your-bucket/todomyday-backups/${BACKUP_TIMESTAMP}/"
```

### Backblaze B2
```bash
# Install
pip install b2

# Configure
b2 authorize-account

# Edit external-backup.sh, uncomment:
b2 sync "${BACKUP_SUBDIR}" "b2://your-bucket/todomyday-backups/${BACKUP_TIMESTAMP}/"
```

### Remote Server (rsync)
```bash
# Setup SSH key authentication first
ssh-copy-id user@backup-server

# Edit external-backup.sh, uncomment:
rsync -avz "${BACKUP_SUBDIR}" user@backup-server:/backups/todomyday/${BACKUP_TIMESTAMP}/
```

---

## Troubleshooting

### Internal backup not running
```bash
sudo docker compose logs backup --tail=50
sudo docker compose restart backup
```

### External backup permission denied
```bash
# Check ownership
ls -la ~/mrbrain/backend/scripts/
# Make executable
chmod +x ~/mrbrain/backend/scripts/external-backup.sh
```

### Disk space issues
```bash
# Check usage
df -h
# Reduce retention in scripts
# Or enable compression in external-backup.sh
```

---

## Additional Resources

- [DATA_RECOVERY_AND_BACKUP.md](../../DATA_RECOVERY_AND_BACKUP.md) - Full documentation
- [BACKUP_QUICK_START.md](../../BACKUP_QUICK_START.md) - Quick setup guide
- [SUPABASE_SETUP.md](../../SUPABASE_SETUP.md) - Supabase configuration
- [CLAUDE.md](../../CLAUDE.md) - Project overview
