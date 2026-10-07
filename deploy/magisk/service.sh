#!/system/bin/sh
# Droidspaces WebUI Background Service (Magisk / KernelSU)
MODDIR=${0%/*}

# Wait for boot completion
until [ "$(getprop sys.boot_completed)" = "1" ]; do
  sleep 3
done
sleep 2

# Port 84 default (can be overridden in config or env)
export DSWEB_PORT=84
export PATH="/data/local/Droidspaces/bin:/data/adb/ksu/bin:$PATH"

if [ -f "$MODDIR/bin/dsweb_arm64" ]; then
  chmod 755 "$MODDIR/bin/dsweb_arm64"
  nohup "$MODDIR/bin/dsweb_arm64" -port 84 > "$MODDIR/daemon.log" 2>&1 &
fi
