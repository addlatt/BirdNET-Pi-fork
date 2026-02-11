import { useState, useEffect, useRef, useCallback } from 'preact/hooks';
import type { JSX } from 'preact';
import { FormSection, AlertMessage, SaveButton } from '../components/settings/FormInputs';
import { downloadBackup, uploadRestore, fetchRestoreStatus } from '../hooks/useApi';
import { useWebSocket } from '../hooks/useWebSocket';
import type { RestoreStatusResponse } from '../types/api';

/**
 * Backup/Restore page component.
 * Download system backups and restore from backup files.
 */
export function Backup(): JSX.Element {
  // Download state
  const [downloading, setDownloading] = useState(false);
  const [downloadError, setDownloadError] = useState<string | null>(null);
  const [downloadSuccess, setDownloadSuccess] = useState(false);

  // Restore state
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [restoreId, setRestoreId] = useState<string | null>(null);
  const [restoreStatus, setRestoreStatus] = useState<RestoreStatusResponse | null>(null);
  const [restoreError, setRestoreError] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // WebSocket for real-time restore progress
  const wsUrl = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws`;
  const { subscribe } = useWebSocket(wsUrl);

  // Subscribe to restore_progress WebSocket messages
  useEffect(() => {
    const unsub = subscribe<RestoreStatusResponse>('restore_progress', (payload) => {
      if (restoreId && payload.id === restoreId) {
        setRestoreStatus(payload);
        if (payload.status === 'completed' || payload.status === 'failed') {
          if (payload.status === 'failed' && payload.error) {
            setRestoreError(payload.error);
          }
          stopPolling();
        }
      }
    });
    return unsub;
  }, [restoreId, subscribe]);

  // Polling fallback for restore progress
  const startPolling = useCallback((id: string) => {
    if (pollIntervalRef.current) return;
    pollIntervalRef.current = setInterval(async () => {
      try {
        const status = await fetchRestoreStatus(id);
        setRestoreStatus(status);
        if (status.status === 'completed' || status.status === 'failed') {
          if (status.status === 'failed' && status.error) {
            setRestoreError(status.error);
          }
          stopPolling();
        }
      } catch {
        // WebSocket will handle it; keep polling
      }
    }, 2000);
  }, []);

  const stopPolling = useCallback(() => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
  }, []);

  // Clean up polling on unmount
  useEffect(() => {
    return () => stopPolling();
  }, [stopPolling]);

  // Handle download
  const handleDownload = async () => {
    setDownloading(true);
    setDownloadError(null);
    setDownloadSuccess(false);
    try {
      await downloadBackup();
      setDownloadSuccess(true);
    } catch (err) {
      setDownloadError(err instanceof Error ? err.message : 'Download failed');
    } finally {
      setDownloading(false);
    }
  };

  // Handle file selection
  const handleFileChange = (e: Event) => {
    const input = e.target as HTMLInputElement;
    setSelectedFile(input.files?.[0] ?? null);
    setRestoreError(null);
  };

  // Handle restore upload
  const handleRestore = async () => {
    if (!selectedFile) return;

    setUploading(true);
    setRestoreError(null);
    setRestoreStatus(null);
    setRestoreId(null);

    try {
      const result = await uploadRestore(selectedFile);
      setRestoreId(result.restore_id);
      setRestoreStatus({
        id: result.restore_id,
        status: 'uploading',
        progress: 0,
        stage: 'Starting restore...',
        started_at: new Date().toISOString(),
      });
      startPolling(result.restore_id);
    } catch (err) {
      setRestoreError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploading(false);
    }
  };

  // Reset restore state
  const handleReset = () => {
    stopPolling();
    setSelectedFile(null);
    setRestoreId(null);
    setRestoreStatus(null);
    setRestoreError(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const isRestoring = restoreStatus !== null &&
    restoreStatus.status !== 'completed' &&
    restoreStatus.status !== 'failed';

  return (
    <div class="max-w-4xl mx-auto p-4 sm:p-6">
      <div class="mb-6">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Backup & Restore</h1>
        <p class="text-gray-500 dark:text-gray-400 mt-1">
          Download a full system backup or restore from a previous backup file.
        </p>
      </div>

      {/* Download Backup Section */}
      <FormSection
        title="Download Backup"
        description="Create and download a complete backup of your BirdNET-Pi data, including configuration, database, recordings, and charts."
      >
        <>
          {downloadError ? (
            <AlertMessage type="error" message={downloadError} onDismiss={() => setDownloadError(null)} />
          ) : downloadSuccess ? (
            <AlertMessage type="success" message="Backup download started." onDismiss={() => setDownloadSuccess(false)} />
          ) : null}
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
            The backup includes your configuration (birdnet.conf), detection database (birds.db),
            species lists, recordings, and chart data. Large installations may take a few minutes to download.
          </p>
          <SaveButton
            onClick={handleDownload}
            saving={downloading}
            text="Download Backup"
            savingText="Preparing..."
          />
        </>
      </FormSection>

      {/* Restore from Backup Section */}
      <FormSection
        title="Restore from Backup"
        description="Upload a previously downloaded backup file to restore your BirdNET-Pi data."
      >
        <>
          {restoreError ? (
            <AlertMessage type="error" message={restoreError} onDismiss={() => setRestoreError(null)} />
          ) : restoreStatus?.status === 'completed' ? (
            <AlertMessage type="success" message="Restore completed successfully. You may need to restart services for changes to take effect." />
          ) : null}

          {/* File Input */}
          {!isRestoring && restoreStatus?.status !== 'completed' ? (
            <div class="space-y-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Backup File
                </label>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".tar.gz,.tgz"
                  onChange={handleFileChange}
                  disabled={uploading}
                  class="block w-full text-sm text-gray-500 dark:text-gray-400
                         file:mr-4 file:py-2 file:px-4
                         file:rounded-lg file:border-0
                         file:text-sm file:font-medium
                         file:bg-primary-50 file:text-primary-700
                         dark:file:bg-primary-900/30 dark:file:text-primary-300
                         file:cursor-pointer
                         hover:file:bg-primary-100 dark:hover:file:bg-primary-900/50
                         disabled:opacity-50 disabled:cursor-not-allowed"
                />
                <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  Accepted formats: .tar.gz, .tgz (max 500MB)
                </p>
              </div>
              <SaveButton
                onClick={handleRestore}
                saving={uploading}
                disabled={!selectedFile}
                text="Upload & Restore"
                savingText="Uploading..."
              />
            </div>
          ) : null}

          {/* Progress Bar */}
          {restoreStatus && restoreStatus.status !== 'failed' ? (
            <div class="space-y-3">
              <div class="flex justify-between text-sm">
                <span class="text-gray-700 dark:text-gray-300 font-medium">
                  {restoreStatus.stage}
                </span>
                <span class="text-gray-500 dark:text-gray-400">
                  {restoreStatus.progress}%
                </span>
              </div>
              <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
                <div
                  class={`h-3 rounded-full transition-all duration-300 ${
                    restoreStatus.status === 'completed'
                      ? 'bg-green-500'
                      : 'bg-primary-600'
                  }`}
                  style={{ width: `${restoreStatus.progress}%` }}
                />
              </div>
              {restoreStatus.status === 'completed' ? (
                <button
                  type="button"
                  onClick={handleReset}
                  class="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
                >
                  Start a new restore
                </button>
              ) : null}
            </div>
          ) : null}
        </>
      </FormSection>

      {/* Warning */}
      <div class="p-4 bg-yellow-50 dark:bg-yellow-900/30 rounded-lg border border-yellow-200 dark:border-yellow-700">
        <h4 class="text-sm font-medium text-yellow-800 dark:text-yellow-200 mb-2">
          Important Notes
        </h4>
        <ul class="text-sm text-yellow-700 dark:text-yellow-300 space-y-1 list-disc list-inside">
          <li>Restoring a backup will overwrite existing data and configuration.</li>
          <li>Services may need to be restarted after a restore.</li>
          <li>Only restore backups from trusted sources.</li>
        </ul>
      </div>
    </div>
  );
}

export default Backup;
