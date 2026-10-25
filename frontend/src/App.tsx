import { useState, useEffect } from 'react';
import './App.css';
import { OpenBLP, GetMipmapPreview, GetMipmapInfo, SaveBLP, ConvertFormat, GenerateMipmaps, RemoveMipmaps, Resize, ApplyImageOperation, ImportImage, ExportPNG, GetSupportedFormats, GetCurrentFileInfo, CloseBLP, SelectFile, SelectSaveFile, SelectSavePNG, SelectImageFile, ReplaceImage, CompositeImage, Undo, Redo, CanUndo, CanRedo } from '../wailsjs/go/main/App';
import { Dropzone } from './components/Dropzone';
import { ImageViewer } from './components/ImageViewer';
import { FormatSelector } from './components/FormatSelector';
import { MipmapPanel } from './components/MipmapPanel';
import { ImageOperations } from './components/ImageOperations';
import { CompositionPanel } from './components/CompositionPanel';
import { Header } from './components/Header';
import { OnFileDrop, OnFileDropOff } from '../wailsjs/runtime/runtime';
import { mipmap } from '../wailsjs/go/models';

interface FileInfo {
  version: number;
  format: number;
  formatName: string;
  width: number;
  height: number;
  mipmapCount: number;
  hasAlpha: boolean;
  fileSize: number;
}

function App() {
  const [fileInfo, setFileInfo] = useState<FileInfo | null>(null);
  const [mipmapInfo, setMipmapInfo] = useState<mipmap.MipmapInfo[]>([]);
  const [previewImage, setPreviewImage] = useState<string>('');
  const [currentMipmapLevel, setCurrentMipmapLevel] = useState(0);
  const [supportedFormats, setSupportedFormats] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>('');
  const [canUndo, setCanUndo] = useState(false);
  const [canRedo, setCanRedo] = useState(false);

  useEffect(() => {
    loadSupportedFormats();
    
    // Set up file drop handler
    OnFileDrop((x, y, paths) => {
      if (paths.length > 0) {
        console.log('Dropped file:', paths[0]);
        handleFileOpen(paths[0]);
      }
    }, true);
    
    return () => {
      OnFileDropOff();
    };
  }, []);

  const loadSupportedFormats = async () => {
    try {
      const formats = await GetSupportedFormats();
      setSupportedFormats(formats);
    } catch (err) {
      console.error('Failed to load supported formats:', err);
    }
  };

  const handleFileOpen = async (filePath: string) => {
    setLoading(true);
    setError('');
    try {
      let info: FileInfo;
      if (filePath.toLowerCase().endsWith('.blp')) {
        info = await OpenBLP(filePath);
      } else {
        await ImportImage(filePath, 3);
        info = await GetCurrentFileInfo();
      }
      setFileInfo(info);
      
      const mipmaps = await GetMipmapInfo();
      setMipmapInfo(mipmaps);
      
      const preview = await GetMipmapPreview(0);
      setPreviewImage(preview);
      setCurrentMipmapLevel(0);
      setCanUndo(false);
      setCanRedo(false);
    } catch (err: any) {
      setError(err.message || 'Failed to open file');
    } finally {
      setLoading(false);
    }
  };

  const handleMipmapLevelChange = async (level: number) => {
    try {
      const preview = await GetMipmapPreview(level);
      setPreviewImage(preview);
      setCurrentMipmapLevel(level);
    } catch (err: any) {
      setError(err.message || 'Failed to load mipmap preview');
    }
  };

  const handleFormatConvert = async (format: number) => {
    if (!fileInfo) return;
    
    setLoading(true);
    try {
      await ConvertFormat(format);
      const info = await GetCurrentFileInfo();
      setFileInfo(info);
      
      const preview = await GetMipmapPreview(currentMipmapLevel);
      setPreviewImage(preview);
    } catch (err: any) {
      setError(err.message || 'Failed to convert format');
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateMipmaps = async (filter: string) => {
    if (!fileInfo) return;
    
    setLoading(true);
    try {
      await GenerateMipmaps(filter);
      const info = await GetCurrentFileInfo();
      setFileInfo(info);
      
      const mipmaps = await GetMipmapInfo();
      setMipmapInfo(mipmaps);
      
      const preview = await GetMipmapPreview(currentMipmapLevel);
      setPreviewImage(preview);
    } catch (err: any) {
      setError(err.message || 'Failed to generate mipmaps');
    } finally {
      setLoading(false);
    }
  };

  const handleRemoveMipmaps = async () => {
    if (!fileInfo) return;
    
    setLoading(true);
    try {
      await RemoveMipmaps();
      const info = await GetCurrentFileInfo();
      setFileInfo(info);
      
      const mipmaps = await GetMipmapInfo();
      setMipmapInfo(mipmaps);
      
      const preview = await GetMipmapPreview(0);
      setPreviewImage(preview);
      setCurrentMipmapLevel(0);
    } catch (err: any) {
      setError(err.message || 'Failed to remove mipmaps');
    } finally {
      setLoading(false);
    }
  };

  const handleImageOperation = async (operation: string, params: any) => {
    if (!fileInfo) return;
    
    setLoading(true);
    try {
      await ApplyImageOperation(operation, params);
      const preview = await GetMipmapPreview(currentMipmapLevel);
      setPreviewImage(preview);
    } catch (err: any) {
      setError(err.message || 'Failed to apply image operation');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (filePath: string) => {
    if (!fileInfo) return;
    
    setLoading(true);
    try {
      await SaveBLP(filePath);
    } catch (err: any) {
      setError(err.message || 'Failed to save BLP file');
    } finally {
      setLoading(false);
    }
  };

  const handleExportPNG = async (filePath: string) => {
    if (!fileInfo) return;
    
    setLoading(true);
    try {
      await ExportPNG(filePath, currentMipmapLevel);
    } catch (err: any) {
      setError(err.message || 'Failed to export PNG');
    } finally {
      setLoading(false);
    }
  };

  const handleSelectFile = async () => {
    try {
      const filePath = await SelectFile();
      if (filePath) {
        await handleFileOpen(filePath);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to select file');
    }
  };

  const handleSelectSaveFile = async () => {
    try {
      const defaultName = 'texture.blp';
      const filePath = await SelectSaveFile(defaultName);
      if (filePath) {
        await handleSave(filePath);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to save file');
    }
  };

  const handleSelectExportFile = async () => {
    try {
      const defaultName = 'texture.png';
      let filePath: string;
      try {
        filePath = await SelectSavePNG(defaultName);
      } catch {
        filePath = await SelectSaveFile(defaultName);
      }
      if (filePath) {
        // Ensure .png extension
        if (!filePath.toLowerCase().endsWith('.png')) {
          filePath += '.png';
        }
        await handleExportPNG(filePath);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to export file');
    }
  };

  const syncHistoryState = async () => {
    setCanUndo(await CanUndo());
    setCanRedo(await CanRedo());
  };

  const refreshAfterEdit = async () => {
    const info = await GetCurrentFileInfo();
    setFileInfo(info);
    const mipmaps = await GetMipmapInfo();
    setMipmapInfo(mipmaps);
    const preview = await GetMipmapPreview(0);
    setPreviewImage(preview);
    setCurrentMipmapLevel(0);
    await syncHistoryState();
  };

  const handleUndo = async () => {
    if (!fileInfo) return;
    setLoading(true);
    try {
      await Undo();
      await refreshAfterEdit();
    } catch (err: any) {
      setError(err.message || 'Failed to undo');
    } finally {
      setLoading(false);
    }
  };

  const handleRedo = async () => {
    if (!fileInfo) return;
    setLoading(true);
    try {
      await Redo();
      await refreshAfterEdit();
    } catch (err: any) {
      setError(err.message || 'Failed to redo');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (!fileInfo || loading) return;
      const mod = e.ctrlKey || e.metaKey;
      if (mod && e.key.toLowerCase() === 'z' && !e.shiftKey) {
        e.preventDefault();
        handleUndo();
      } else if (mod && (e.key.toLowerCase() === 'y' || (e.key.toLowerCase() === 'z' && e.shiftKey))) {
        e.preventDefault();
        handleRedo();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [fileInfo, loading, handleUndo, handleRedo]);

  const handleReplaceImage = async () => {
    if (!fileInfo) return;
    try {
      const filePath = await SelectImageFile();
      if (!filePath) return;
      setLoading(true);
      await ReplaceImage(filePath);
      await refreshAfterEdit();
    } catch (err: any) {
      setError(err.message || 'Failed to replace image');
    } finally {
      setLoading(false);
    }
  };

  const handleOverlayImage = async (opacity: number, fit: boolean) => {
    if (!fileInfo) return;
    try {
      const filePath = await SelectImageFile();
      if (!filePath) return;
      setLoading(true);
      await CompositeImage(filePath, opacity / 100, fit);
      await refreshAfterEdit();
    } catch (err: any) {
      setError(err.message || 'Failed to overlay image');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    CloseBLP();
    setFileInfo(null);
    setMipmapInfo([]);
    setPreviewImage('');
    setCurrentMipmapLevel(0);
    setError('');
    setCanUndo(false);
    setCanRedo(false);
  };

  return (
    <div className="app">
      <Header 
        fileInfo={fileInfo}
        onSave={handleSelectSaveFile}
        onExport={handleSelectExportFile}
        onClose={handleClose}
        onSelectFile={handleSelectFile}
        onUndo={handleUndo}
        onRedo={handleRedo}
        canUndo={canUndo}
        canRedo={canRedo}
      />
      
      <div className="main-content">
        {!fileInfo ? (
          <div className="welcome-screen">
            <div className="welcome-content">
              <h1>BLP Editor</h1>
              <p>World of Warcraft Texture Editor</p>
              <Dropzone onFileOpen={handleFileOpen} />
            </div>
          </div>
        ) : (
          <div className="editor-layout">
            <div className="viewer-section">
              <ImageViewer 
                image={previewImage}
                fileInfo={fileInfo}
                loading={loading}
              />
            </div>
            
            <div className="sidebar">
              <FormatSelector 
                currentFormat={fileInfo.format}
                formats={supportedFormats}
                onConvert={handleFormatConvert}
                disabled={loading}
              />
              
              <MipmapPanel 
                mipmapInfo={mipmapInfo}
                currentLevel={currentMipmapLevel}
                onLevelChange={handleMipmapLevelChange}
                onGenerateMipmaps={handleGenerateMipmaps}
                onRemoveMipmaps={handleRemoveMipmaps}
                disabled={loading}
              />
              
              <ImageOperations 
                onOperation={handleImageOperation}
                disabled={loading}
              />

              <CompositionPanel
                disabled={loading}
                onReplace={handleReplaceImage}
                onOverlay={handleOverlayImage}
              />
            </div>
          </div>
        )}
        
        {error && (
          <div className="error-banner">
            <span>{error}</span>
            <button onClick={() => setError('')}>×</button>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;
