#!/bin/bash
# External backup script - copies database backups from Docker volume to external location
# Run this on the host machine (not inside Docker)
# This protects against Docker volume corruption/deletion
# Usage: ./external-backup.sh [optional-external-path]

set -e

# Configuration
# Find the project directory (where docker-compose.yml is located)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DATA_DIR="${PROJECT_DIR}/data"
EXTERNAL_BACKUP_DIR="${1:-${HOME}/backups/todomyday}"
RETENTION_DAYS=30

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1"
}

# Create external backup directory
mkdir -p "${EXTERNAL_BACKUP_DIR}"

log_info "Starting external backup process..."
log_info "Source: ${DATA_DIR}"
log_info "Destination: ${EXTERNAL_BACKUP_DIR}"

# Check if data directory exists
if [ ! -d "${DATA_DIR}" ]; then
    log_error "Data directory not found: ${DATA_DIR}"
    exit 1
fi

# Create timestamped backup subdirectory
BACKUP_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_SUBDIR="${EXTERNAL_BACKUP_DIR}/${BACKUP_TIMESTAMP}"
mkdir -p "${BACKUP_SUBDIR}"

# Copy main database
if [ -f "${DATA_DIR}/todomyday.db" ]; then
    log_info "Copying main database..."
    cp "${DATA_DIR}/todomyday.db" "${BACKUP_SUBDIR}/todomyday.db"
    DB_SIZE=$(du -h "${BACKUP_SUBDIR}/todomyday.db" | cut -f1)
    log_info "✅ Main database copied (${DB_SIZE})"
else
    log_warn "Main database not found"
fi

# Copy WAL file if exists
if [ -f "${DATA_DIR}/todomyday.db-wal" ]; then
    cp "${DATA_DIR}/todomyday.db-wal" "${BACKUP_SUBDIR}/todomyday.db-wal"
    WAL_SIZE=$(du -h "${BACKUP_SUBDIR}/todomyday.db-wal" | cut -f1)
    log_info "✅ WAL file copied (${WAL_SIZE})"
fi

# Copy automated backups directory
if [ -d "${DATA_DIR}/backups" ]; then
    log_info "Copying automated backups..."
    cp -r "${DATA_DIR}/backups" "${BACKUP_SUBDIR}/"
    BACKUPS_COUNT=$(find "${BACKUP_SUBDIR}/backups" -name "*.db" -type f | wc -l)
    log_info "✅ Automated backups copied (${BACKUPS_COUNT} files)"
fi

# Copy pre-migration backups
PRE_MIGRATION_COUNT=$(find "${DATA_DIR}" -maxdepth 1 -name "todomyday-pre-migration-*.db" -type f | wc -l)
if [ ${PRE_MIGRATION_COUNT} -gt 0 ]; then
    log_info "Copying pre-migration backups..."
    find "${DATA_DIR}" -maxdepth 1 -name "todomyday-pre-migration-*.db" -type f -exec cp {} "${BACKUP_SUBDIR}/" \;
    log_info "✅ Pre-migration backups copied (${PRE_MIGRATION_COUNT} files)"
fi

# Copy recovery backups
RECOVERY_COUNT=$(find "${DATA_DIR}" -maxdepth 1 -name "todomyday-recovered-*.db" -type f | wc -l)
if [ ${RECOVERY_COUNT} -gt 0 ]; then
    log_info "Copying recovery backups..."
    find "${DATA_DIR}" -maxdepth 1 -name "todomyday-recovered-*.db" -type f -exec cp {} "${BACKUP_SUBDIR}/" \;
    log_info "✅ Recovery backups copied (${RECOVERY_COUNT} files)"
fi

# Create backup manifest
MANIFEST_FILE="${BACKUP_SUBDIR}/backup-manifest.txt"
cat > "${MANIFEST_FILE}" << EOF
Backup Manifest
===============
Timestamp: $(date '+%Y-%m-%d %H:%M:%S')
Source: ${DATA_DIR}
Destination: ${BACKUP_SUBDIR}

Files Backed Up:
EOF

find "${BACKUP_SUBDIR}" -type f -exec ls -lh {} \; | awk '{print $9, "(" $5 ")"}' >> "${MANIFEST_FILE}"

# Calculate total backup size
TOTAL_SIZE=$(du -sh "${BACKUP_SUBDIR}" | cut -f1)
echo "" >> "${MANIFEST_FILE}"
echo "Total Backup Size: ${TOTAL_SIZE}" >> "${MANIFEST_FILE}"

log_info "✅ Backup manifest created"

# Compress backup (optional - uncomment if you want compression)
# log_info "Compressing backup..."
# tar -czf "${EXTERNAL_BACKUP_DIR}/backup_${BACKUP_TIMESTAMP}.tar.gz" -C "${EXTERNAL_BACKUP_DIR}" "${BACKUP_TIMESTAMP}"
# rm -rf "${BACKUP_SUBDIR}"
# COMPRESSED_SIZE=$(du -h "${EXTERNAL_BACKUP_DIR}/backup_${BACKUP_TIMESTAMP}.tar.gz" | cut -f1)
# log_info "✅ Backup compressed (${COMPRESSED_SIZE})"

# Clean up old external backups (keep last X days)
log_info "Cleaning up old backups (keeping last ${RETENTION_DAYS} days)..."
find "${EXTERNAL_BACKUP_DIR}" -maxdepth 1 -type d -name "20*" -mtime +${RETENTION_DAYS} -exec rm -rf {} \;
# Uncomment if using compression:
# find "${EXTERNAL_BACKUP_DIR}" -maxdepth 1 -name "backup_*.tar.gz" -mtime +${RETENTION_DAYS} -delete

REMAINING_BACKUPS=$(find "${EXTERNAL_BACKUP_DIR}" -maxdepth 1 -type d -name "20*" | wc -l)
log_info "✅ Old backups cleaned up (${REMAINING_BACKUPS} backups remaining)"

# Summary
echo ""
log_info "================================"
log_info "External Backup Complete!"
log_info "================================"
log_info "Backup Location: ${BACKUP_SUBDIR}"
log_info "Total Size: ${TOTAL_SIZE}"
log_info "Retention: ${RETENTION_DAYS} days"
echo ""

# Optional: Upload to cloud storage (uncomment and configure as needed)
# log_info "Uploading to cloud storage..."
#
# # Example: AWS S3
# # aws s3 sync "${BACKUP_SUBDIR}" "s3://your-bucket/todomyday-backups/${BACKUP_TIMESTAMP}/"
#
# # Example: Google Cloud Storage
# # gsutil -m rsync -r "${BACKUP_SUBDIR}" "gs://your-bucket/todomyday-backups/${BACKUP_TIMESTAMP}/"
#
# # Example: Backblaze B2
# # b2 sync "${BACKUP_SUBDIR}" "b2://your-bucket/todomyday-backups/${BACKUP_TIMESTAMP}/"
#
# # Example: rsync to remote server
# # rsync -avz "${BACKUP_SUBDIR}" user@remote-server:/backups/todomyday/${BACKUP_TIMESTAMP}/
#
# log_info "✅ Cloud backup complete"

exit 0
