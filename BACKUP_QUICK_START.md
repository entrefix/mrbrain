# Backup System Quick Start Guide

## 🚨 FIRST: Recover Your Existing Data

Before setting up backups, recover your current data:

```bash
cd ~/mrbrain
sudo docker compose stop backend
sudo docker compose run --rm backend sqlite3 /data/todomyday.db "PRAGMA wal_checkpoint(TRUNCATE);"
sudo docker compose run --rm backend sqlite3 /data/todomyday.db "SELECT COUNT(*) FROM memories;"
sudo docker compose run --rm backend sqlite3 /data/todomyday.db ".backup /data/todomyday-recovered-$(date +%Y%m%d).db"
sudo docker compose start backend
```

---

## 3-Tier Backup Strategy

### Tier 1: Internal Automated Backups (Inside Docker)
**Purpose**: Quick recovery, frequent backups
**Frequency**: Every 2 hours
**Retention**: 7 days
**Setup**: Automatic (already configured in docker-compose.yml)

### Tier 2: External Backups (Outside Docker Volume)
**Purpose**: Protection against volume corruption
**Frequency**: Daily
**Retention**: 30 days
**Setup**: Required (see below)

### Tier 3: Cloud/Remote Backups (Off-Site)
**Purpose**: Disaster recovery
**Frequency**: Daily/Weekly
**Retention**: Configurable
**Setup**: Optional (see below)

---

## Quick Setup (5 Minutes)

### Step 1: Deploy Latest Code

```bash
cd ~/mrbrain
git pull
sudo docker compose up --build -d
```

Verify all services running:
```bash
sudo docker compose ps
# Should show: backend, frontend, backup (all running)
```

### Step 2: Setup External Backups

Create backup directory:
```bash
mkdir -p ~/backups/todomyday
```

Make script executable (if not already):
```bash
chmod +x ~/mrbrain/backend/scripts/external-backup.sh
```

Test external backup:
```bash
cd ~/mrbrain/backend/scripts
./external-backup.sh
```

Verify backup created:
```bash
ls -lh ~/backups/todomyday/
```

### Step 3: Automate External Backups

Add to crontab:
```bash
crontab -e
```

Add this line (daily at 3 AM):
```
0 3 * * * cd ~/mrbrain/backend/scripts && ./external-backup.sh >> ~/backups/external-backup.log 2>&1
```

Save and exit. Verify cron job added:
```bash
crontab -l
```

---

## Verification Checklist

- [ ] Docker backup service is running: `sudo docker compose ps backup`
- [ ] Internal backups directory exists: `ls ~/mrbrain/data/backups/`
- [ ] Internal backup log has entries: `cat ~/mrbrain/data/backups/backup.log`
- [ ] External backup directory exists: `ls ~/backups/todomyday/`
- [ ] External backup cron job added: `crontab -l | grep external-backup`
- [ ] Health checks appear in logs: `sudo docker compose logs backend | grep "health check"`

---

## Monitoring Commands

```bash
# Check internal backup status
sudo docker compose exec backend cat /data/backups/backup.log | tail -20

# Check external backup status
cat ~/backups/external-backup.log | tail -20

# List all backups
ls -lh ~/mrbrain/data/backups/
ls -lh ~/backups/todomyday/

# Check disk space
df -h ~/mrbrain/data
df -h ~/backups
```

---

## Optional: Cloud Backups

### AWS S3

```bash
# Install AWS CLI
sudo apt install awscli

# Configure credentials
aws configure

# Edit external-backup.sh and uncomment S3 section:
nano ~/mrbrain/backend/scripts/external-backup.sh
# Uncomment: aws s3 sync "${BACKUP_SUBDIR}" "s3://your-bucket/..."
```

### Backblaze B2 (Cheaper Alternative)

```bash
# Install B2 CLI
pip install b2

# Authenticate
b2 authorize-account

# Edit external-backup.sh and uncomment B2 section
```

---

## Emergency Recovery

### If Docker volume is corrupted:

```bash
# 1. Stop backend
cd ~/mrbrain
sudo docker compose stop backend

# 2. List external backups
ls -lh ~/backups/todomyday/

# 3. Copy latest backup
cp ~/backups/todomyday/[LATEST_TIMESTAMP]/todomyday.db ~/mrbrain/data/

# 4. Start backend
sudo docker compose start backend

# 5. Verify
sudo docker compose exec backend sqlite3 /data/todomyday.db "SELECT COUNT(*) FROM memories;"
```

### If everything is lost but you have cloud backups:

```bash
# Download from S3
aws s3 sync s3://your-bucket/todomyday-backups/[TIMESTAMP]/ ~/mrbrain/data/

# Or from B2
b2 sync b2://your-bucket/todomyday-backups/[TIMESTAMP]/ ~/mrbrain/data/

# Then start services
sudo docker compose start backend
```

---

## Best Practices

1. ✅ **Test restores monthly** - Don't wait for disaster to test
2. ✅ **Monitor disk space** - Ensure backup drives have space
3. ✅ **Verify backup logs** - Check weekly for errors
4. ✅ **Keep 3-2-1 rule**: 3 copies, 2 different media, 1 off-site
5. ✅ **Document procedures** - Keep this guide updated

---

## Troubleshooting

**Internal backups not running?**
```bash
sudo docker compose logs backup
sudo docker compose restart backup
```

**External backup fails?**
```bash
# Check permissions
ls -la ~/backups/
# Check disk space
df -h
# Run manually with verbose output
cd ~/mrbrain/backend/scripts
bash -x ./external-backup.sh
```

**Need more space?**
```bash
# Reduce internal backup retention (edit backup.sh)
# Change: find ... -mtime +7 ... to -mtime +3

# Reduce external backup retention (edit external-backup.sh)
# Change: RETENTION_DAYS=30 to RETENTION_DAYS=14

# Enable compression in external-backup.sh
# Uncomment the tar.gz section
```

---

## Summary

You now have:
- ✅ Automated internal backups (every 2 hours, 7 days retention)
- ✅ External backups outside Docker volume (daily, 30 days retention)
- ✅ Database health checks on startup
- ✅ Safe database migration code
- ✅ Optional cloud backup capability

**Your data is protected!** 🎉

For detailed documentation, see [DATA_RECOVERY_AND_BACKUP.md](DATA_RECOVERY_AND_BACKUP.md)
