import { FileText, FolderOpen, Save, Download, X, Undo2, Redo2 } from 'lucide-react';

interface HeaderProps {
  fileInfo: any;
  onClose: () => void;
  onSelectFile: () => void;
  onSave: () => void;
  onExport: () => void;
  onUndo: () => void;
  onRedo: () => void;
  canUndo: boolean;
  canRedo: boolean;
}

export function Header({ fileInfo, onClose, onSelectFile, onSave, onExport, onUndo, onRedo, canUndo, canRedo }: HeaderProps) {
  return (
    <header className="header">
      <div className="header-left">
        <FileText className="logo-icon" />
        <h1>BLP Editor</h1>
      </div>
      
      <div className="header-center">
        {fileInfo && (
          <div className="file-info">
            <span>{fileInfo.width}x{fileInfo.height}</span>
            <span className="separator">|</span>
            <span>{fileInfo.formatName}</span>
            <span className="separator">|</span>
            <span>{fileInfo.mipmapCount} mipmaps</span>
          </div>
        )}
      </div>
      
      <div className="header-right">
        {!fileInfo && (
          <button 
            className="header-btn"
            onClick={onSelectFile}
            title="Open File"
          >
            <FolderOpen size={18} />
            Open
          </button>
        )}
        
        {fileInfo && (
          <>
            <button
              className="header-btn"
              onClick={onUndo}
              title="Undo (Ctrl+Z)"
              disabled={!canUndo}
            >
              <Undo2 size={18} />
            </button>

            <button
              className="header-btn"
              onClick={onRedo}
              title="Redo (Ctrl+Y)"
              disabled={!canRedo}
            >
              <Redo2 size={18} />
            </button>

            <button 
              className="header-btn"
              onClick={onSave}
              title="Save BLP"
            >
              <Save size={18} />
              Save
            </button>
            
            <button 
              className="header-btn"
              onClick={onExport}
              title="Export as PNG"
            >
              <Download size={18} />
              Export PNG
            </button>
            
            <button 
              className="header-btn close-btn"
              onClick={onClose}
              title="Close file"
            >
              <X size={18} />
            </button>
          </>
        )}
      </div>
    </header>
  );
}
