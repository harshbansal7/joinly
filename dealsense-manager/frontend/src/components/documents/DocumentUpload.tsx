/**
 * Document upload component for pitch decks and other startup documents
 */

'use client';

import { useState, useCallback } from 'react';
import { Upload, FileText, Loader2, CheckCircle, XCircle, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { documentsApi, Document } from '@/lib/api';

interface DocumentUploadProps {
  agentId: string;
  documents: Document[];
  onDocumentsChange: () => void;
}

export function DocumentUpload({ agentId, documents, onDocumentsChange }: DocumentUploadProps) {
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<string>('');
  const [dragActive, setDragActive] = useState(false);

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFile(e.dataTransfer.files[0]);
    }
  }, [agentId]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.preventDefault();
    if (e.target.files && e.target.files[0]) {
      handleFile(e.target.files[0]);
    }
  };

  const handleFile = async (file: File) => {
    // Validate file type
    const allowedTypes = [
      'application/pdf',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    ];

    if (!allowedTypes.includes(file.type)) {
      alert('Please upload a PDF, DOCX, or PPTX file');
      return;
    }

    // Validate file size (50MB max)
    if (file.size > 50 * 1024 * 1024) {
      alert('File size must be less than 50MB');
      return;
    }

    setUploading(true);
    setUploadProgress(`Uploading ${file.name}...`);

    try {
      await documentsApi.upload(agentId, file);
      setUploadProgress('Processing document...');
      
      // Wait a bit for processing to start
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      onDocumentsChange();
      setUploadProgress('');
    } catch (error) {
      console.error('Upload failed:', error);
      alert('Failed to upload document. Please try again.');
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async (documentId: string) => {
    if (!confirm('Are you sure you want to delete this document?')) {
      return;
    }

    try {
      await documentsApi.delete(documentId);
      onDocumentsChange();
    } catch (error) {
      console.error('Delete failed:', error);
      alert('Failed to delete document');
    }
  };

  const formatFileSize = (bytes: number) => {
    if (!bytes || bytes === 0) return '0 Bytes';
    if (typeof bytes !== 'number' || isNaN(bytes)) return 'Unknown';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'processed':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
      case 'processing':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300';
      case 'failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'processed':
        return <CheckCircle className="h-4 w-4" />;
      case 'processing':
        return <Loader2 className="h-4 w-4 animate-spin" />;
      case 'failed':
        return <XCircle className="h-4 w-4" />;
      default:
        return <FileText className="h-4 w-4" />;
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <FileText className="h-5 w-5" />
          Documents
        </CardTitle>
        <CardDescription>
          Upload pitch decks, business plans, and other startup documents for AI-powered analysis
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Upload Area */}
        <div
          className={`relative border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
            dragActive
              ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/20'
              : 'border-gray-300 dark:border-gray-700 hover:border-gray-400 dark:hover:border-gray-600'
          }`}
          onDragEnter={handleDrag}
          onDragLeave={handleDrag}
          onDragOver={handleDrag}
          onDrop={handleDrop}
        >
          <input
            type="file"
            id="file-upload"
            className="hidden"
            accept=".pdf,.docx,.pptx"
            onChange={handleChange}
            disabled={uploading}
          />
          
          <div className="flex flex-col items-center gap-3">
            {uploading ? (
              <>
                <Loader2 className="h-12 w-12 text-blue-600 animate-spin" />
                <p className="text-sm text-gray-600 dark:text-gray-400">{uploadProgress}</p>
              </>
            ) : (
              <>
                <Upload className="h-12 w-12 text-gray-400" />
                <div>
                  <p className="text-base font-medium text-gray-900 dark:text-white">
                    Drop your files here, or{' '}
                    <label htmlFor="file-upload" className="text-blue-600 hover:text-blue-700 cursor-pointer">
                      browse
                    </label>
                  </p>
                  <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                    Supports PDF, DOCX, PPTX up to 50MB
                  </p>
                </div>
              </>
            )}
          </div>
        </div>

        {/* Document List */}
        {documents.length > 0 && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium text-gray-900 dark:text-white">
              Uploaded Documents ({documents.length})
            </h3>
            <div className="space-y-2">
              {documents.map((doc) => (
                <div
                  key={doc.id}
                  className="flex items-center justify-between p-3 border border-gray-200 dark:border-gray-800 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-900/50 transition-colors"
                >
                  <div className="flex items-center gap-3 flex-1 min-w-0">
                    <div className="flex-shrink-0">
                      {getStatusIcon(doc.status)}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-900 dark:text-white truncate">
                        {doc.name}
                      </p>
                      <div className="flex items-center gap-2 mt-1">
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                          {formatFileSize(doc.file_size)}
                        </p>
                        {doc.page_count > 0 && (
                          <>
                            <span className="text-gray-300 dark:text-gray-700">•</span>
                            <p className="text-xs text-gray-500 dark:text-gray-400">
                              {doc.page_count} pages
                            </p>
                          </>
                        )}
                        {doc.processed_at && (
                          <>
                            <span className="text-gray-300 dark:text-gray-700">•</span>
                            <p className="text-xs text-gray-500 dark:text-gray-400">
                              Processed {new Date(doc.processed_at).toLocaleDateString()}
                            </p>
                          </>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge className={`${getStatusColor(doc.status)} text-xs`}>
                      {doc.status}
                    </Badge>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDelete(doc.id)}
                      className="h-8 w-8 p-0"
                      disabled={doc.status === 'processing'}
                    >
                      <Trash2 className="h-4 w-4 text-red-600" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {documents.length === 0 && !uploading && (
          <div className="text-center py-6">
            <FileText className="h-12 w-12 text-gray-400 mx-auto mb-3" />
            <p className="text-sm text-gray-600 dark:text-gray-400">
              No documents uploaded yet. Upload a pitch deck to get started!
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

