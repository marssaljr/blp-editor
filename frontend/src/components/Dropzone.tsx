import { useCallback } from 'react';
import { useDropzone } from 'react-dropzone';
import { Upload, FileImage } from 'lucide-react';

interface DropzoneProps {
  onFileOpen: (filePath: string) => void;
}

export function Dropzone({ onFileOpen }: DropzoneProps) {
  const onDrop = useCallback((acceptedFiles: File[]) => {
    if (acceptedFiles.length > 0) {
      const file = acceptedFiles[0] as any;
      if (file.path) {
        onFileOpen(file.path);
      } else {
        alert('File path not available. Use the Open button to select files.');
      }
    }
  }, [onFileOpen]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    noClick: false,
    accept: {
      'image/*': ['.blp', '.png', '.jpg', '.jpeg']
    },
    multiple: false
  });

  return (
    <div {...getRootProps()} className={`dropzone ${isDragActive ? 'active' : ''}`}>
      <input {...getInputProps()} />
      <div className="dropzone-content">
        {isDragActive ? (
          <>
            <Upload className="dropzone-icon" size={48} />
            <p>Drop the file here...</p>
          </>
        ) : (
          <>
            <FileImage className="dropzone-icon" size={48} />
            <p>Drag & drop or click to select a file</p>
            <p className="dropzone-hint">Supports .blp, .png, .jpg, .jpeg</p>
          </>
        )}
      </div>
    </div>
  );
}
