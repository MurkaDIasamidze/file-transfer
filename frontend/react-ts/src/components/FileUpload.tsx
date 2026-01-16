import React, { useState } from 'react';
import { FileUploadService, UploadProgress } from '../services/uploadService';

export const FileUpload: React.FC = () => {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [progress, setProgress] = useState<UploadProgress | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      setSelectedFile(file);
      setProgress(null);
      setError(null);
      setSuccess(false);
    }
  };

  const handleUpload = async () => {
    if (!selectedFile) return;

    setIsUploading(true);
    setError(null);
    setSuccess(false);

    const uploadService = new FileUploadService((prog) => {
      setProgress(prog);
    });

    try {
      await uploadService.uploadFile(selectedFile);
      setSuccess(true);
    } catch (err: any) {
      setError(err.message || 'Upload failed');
      console.error('Upload error:', err);
    } finally {
      setIsUploading(false);
    }
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
  };

  return (
    <div style={styles.container}>
      <div style={styles.card}>
        <h1 style={styles.title}>File Transfer System</h1>
        <p style={styles.subtitle}>Upload files with chunked transfer and checksum verification</p>

        <div style={styles.uploadSection}>
          <input
            type="file"
            onChange={handleFileSelect}
            disabled={isUploading}
            style={styles.fileInput}
            id="file-input"
          />
          <label htmlFor="file-input" style={styles.fileLabel}>
            {selectedFile ? selectedFile.name : 'Choose a file'}
          </label>

          {selectedFile && (
            <div style={styles.fileInfo}>
              <p style={styles.fileDetail}>
                <strong>File:</strong> {selectedFile.name}
              </p>
              <p style={styles.fileDetail}>
                <strong>Type:</strong> {selectedFile.type || 'Unknown'}
              </p>
              <p style={styles.fileDetail}>
                <strong>Size:</strong> {formatFileSize(selectedFile.size)}
              </p>
            </div>
          )}

          <button
            onClick={handleUpload}
            disabled={!selectedFile || isUploading}
            style={{
              ...styles.button,
              ...((!selectedFile || isUploading) && styles.buttonDisabled),
            }}
          >
            {isUploading ? 'Uploading...' : 'Upload File'}
          </button>
        </div>

        {progress && (
          <div style={styles.progressSection}>
            <div style={styles.progressBar}>
              <div
                style={{
                  ...styles.progressFill,
                  width: `${progress.percentage}%`,
                }}
              />
            </div>
            <p style={styles.progressText}>
              {progress.percentage}% - Chunk {progress.uploadedChunks} of{' '}
              {progress.totalChunks}
            </p>
            <p style={styles.statusText}>Status: {progress.status}</p>
          </div>
        )}

        {error && (
          <div style={styles.errorBox}>
            <p style={styles.errorText}>❌ Error: {error}</p>
          </div>
        )}

        {success && (
          <div style={styles.successBox}>
            <p style={styles.successText}>
              ✅ File uploaded successfully with verified checksum!
            </p>
          </div>
        )}

        <div style={styles.features}>
          <h3 style={styles.featuresTitle}>Features:</h3>
          <ul style={styles.featuresList}>
            <li>✓ Chunked upload (1MB chunks)</li>
            <li>✓ SHA256 checksum verification</li>
            <li>✓ Automatic retry on failure</li>
            <li>✓ Progress tracking</li>
            <li>✓ Original filename preservation</li>
          </ul>
        </div>
      </div>
    </div>
  );
};

const styles = {
  container: {
    minHeight: '100vh',
    backgroundColor: '#f5f5f5',
    padding: '40px 20px',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
  },
  card: {
    maxWidth: '600px',
    margin: '0 auto',
    backgroundColor: 'white',
    borderRadius: '12px',
    padding: '40px',
    boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
  },
  title: {
    fontSize: '32px',
    fontWeight: 'bold',
    color: '#333',
    marginBottom: '8px',
    textAlign: 'center' as const,
  },
  subtitle: {
    fontSize: '16px',
    color: '#666',
    marginBottom: '32px',
    textAlign: 'center' as const,
  },
  uploadSection: {
    marginBottom: '32px',
  },
  fileInput: {
    display: 'none',
  },
  fileLabel: {
    display: 'block',
    padding: '16px',
    backgroundColor: '#f0f0f0',
    border: '2px dashed #ccc',
    borderRadius: '8px',
    textAlign: 'center' as const,
    cursor: 'pointer',
    marginBottom: '16px',
    transition: 'all 0.3s',
  },
  fileInfo: {
    backgroundColor: '#f9f9f9',
    padding: '16px',
    borderRadius: '8px',
    marginBottom: '16px',
  },
  fileDetail: {
    margin: '8px 0',
    fontSize: '14px',
    color: '#555',
  },
  button: {
    width: '100%',
    padding: '14px',
    backgroundColor: '#007bff',
    color: 'white',
    border: 'none',
    borderRadius: '8px',
    fontSize: '16px',
    fontWeight: 'bold',
    cursor: 'pointer',
    transition: 'background-color 0.3s',
  },
  buttonDisabled: {
    backgroundColor: '#ccc',
    cursor: 'not-allowed',
  },
  progressSection: {
    marginBottom: '24px',
  },
  progressBar: {
    width: '100%',
    height: '24px',
    backgroundColor: '#e0e0e0',
    borderRadius: '12px',
    overflow: 'hidden',
    marginBottom: '12px',
  },
  progressFill: {
    height: '100%',
    backgroundColor: '#28a745',
    transition: 'width 0.3s',
  },
  progressText: {
    textAlign: 'center' as const,
    fontSize: '16px',
    fontWeight: 'bold',
    color: '#333',
    marginBottom: '4px',
  },
  statusText: {
    textAlign: 'center' as const,
    fontSize: '14px',
    color: '#666',
  },
  errorBox: {
    backgroundColor: '#fee',
    padding: '16px',
    borderRadius: '8px',
    marginBottom: '24px',
  },
  errorText: {
    color: '#c00',
    margin: 0,
    fontSize: '14px',
  },
  successBox: {
    backgroundColor: '#efe',
    padding: '16px',
    borderRadius: '8px',
    marginBottom: '24px',
  },
  successText: {
    color: '#0a0',
    margin: 0,
    fontSize: '14px',
    fontWeight: 'bold',
  },
  features: {
    marginTop: '32px',
    paddingTop: '24px',
    borderTop: '1px solid #eee',
  },
  featuresTitle: {
    fontSize: '18px',
    fontWeight: 'bold',
    color: '#333',
    marginBottom: '12px',
  },
  featuresList: {
    listStyle: 'none',
    padding: 0,
    margin: 0,
  },
};