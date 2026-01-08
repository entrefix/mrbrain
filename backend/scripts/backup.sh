#!/bin/sh
# Automated database backup script for Docker container
# Runs daily via docker-compose backup service or cron

BACKUP_DIR="/data/backups"
DATE=$(date +%Y%m%d_%H%M%S)
DB_PATH="/data/todomyday.db"
BACKUP_FILE="${BACKUP_DIR}/todomyday_${DATE}.db"
LOG_FILE="${BACKUP_DIR}/backup.log"

# Create backup directory if it doesn't exist
mkdir -p ${BACKUP_DIR}

# Log start
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting backup..." >> ${LOG_FILE}

# Check if database exists
if [ ! -f "${DB_PATH}" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: Database not found at ${DB_PATH}" >> ${LOG_FILE}
    exit 1
fi

# Checkpoint WAL before backup to ensure consistency
sqlite3 ${DB_PATH} "PRAGMA wal_checkpoint(PASSIVE);" >> ${LOG_FILE} 2>&1

# Create backup using SQLite's backup command (safest method)
if sqlite3 ${DB_PATH} ".backup ${BACKUP_FILE}"; then
    BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✅ Backup created: ${BACKUP_FILE} (${BACKUP_SIZE})" >> ${LOG_FILE}

    # Verify backup integrity
    if sqlite3 ${BACKUP_FILE} "PRAGMA integrity_check;" | grep -q "ok"; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✅ Backup integrity verified" >> ${LOG_FILE}
    else
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] ⚠️  WARNING: Backup integrity check failed" >> ${LOG_FILE}
    fi
else
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ❌ ERROR: Backup failed" >> ${LOG_FILE}
    exit 1
fi

# Keep only last 7 days of backups
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Cleaning up old backups (keeping last 7 days)..." >> ${LOG_FILE}
find ${BACKUP_DIR} -name "todomyday_*.db" -mtime +7 -type f -delete

# Log summary
BACKUP_COUNT=$(find ${BACKUP_DIR} -name "todomyday_*.db" | wc -l)
TOTAL_SIZE=$(du -sh ${BACKUP_DIR} | cut -f1)
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Total backups: ${BACKUP_COUNT}, Total size: ${TOTAL_SIZE}" >> ${LOG_FILE}
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Backup complete!" >> ${LOG_FILE}
echo "" >> ${LOG_FILE}
