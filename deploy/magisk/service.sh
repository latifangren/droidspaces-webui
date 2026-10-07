#!/system/bin/sh
# Droidspaces WebUI Background Service (Magisk / KernelSU / APatch)
MODDIR=${0%/*}

# Wait for Android boot completion
until [ "$(getprop sys.boot_completed)" = "1" ]; do
  sleep 3
done
sleep 2

# Port configuration (fallback to 84)
PORT=84
if [ -f "$MODDIR/port" ]; then
  CONF_PORT=$(cat "$MODDIR/port" | tr -d '\r\n ')
  [ -n "$CONF_PORT" ] && PORT=$CONF_PORT
elif [ -n "$DSWEB_PORT" ]; then
  PORT=$DSWEB_PORT
fi

export DSWEB_PORT=$PORT
export PATH="/data/local/Droidspaces/bin:/data/adb/ksu/bin:/data/adb/magisk:/system/bin:/system/xbin:$PATH"

# Kill existing instance if running
pkill -f "$MODDIR/bin/dsweb_arm64" 2>/dev/null || true

# Log rotation if > 1MB
if [ -f "$MODDIR/daemon.log" ]; then
  LOG_SIZE=$(wc -c < "$MODDIR/daemon.log" 2>/dev/null || echo 0)
  if [ "$LOG_SIZE" -gt 1048576 ]; then
    mv "$MODDIR/daemon.log" "$MODDIR/daemon.log.old"
  fi
fi

if [ -f "$MODDIR/bin/dsweb_arm64" ]; then
  chmod 755 "$MODDIR/bin/dsweb_arm64"
  nohup "$MODDIR/bin/dsweb_arm64" -port "$PORT" > "$MODDIR/daemon.log" 2>&1 &
fi
