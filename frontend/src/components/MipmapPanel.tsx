import { Layers, Plus, Trash2 } from 'lucide-react';
import { mipmap } from '../../wailsjs/go/models';

interface MipmapPanelProps {
  mipmapInfo: mipmap.MipmapInfo[];
  currentLevel: number;
  onLevelChange: (level: number) => void;
  onGenerateMipmaps: (filter: string) => void;
  onRemoveMipmaps: () => void;
  disabled: boolean;
}

export function MipmapPanel({ 
  mipmapInfo, 
  currentLevel, 
  onLevelChange, 
  onGenerateMipmaps, 
  onRemoveMipmaps,
  disabled 
}: MipmapPanelProps) {
  return (
    <div className="panel">
      <div className="panel-header">
        <Layers size={18} />
        <h3>Mipmaps</h3>
      </div>
      <div className="panel-content">
        <div className="mipmap-actions">
          <button
            onClick={() => onGenerateMipmaps('linear')}
            disabled={disabled}
            className="action-btn"
          >
            <Plus size={16} />
            Generate
          </button>
          <button
            onClick={onRemoveMipmaps}
            disabled={disabled || mipmapInfo.length <= 1}
            className="action-btn danger"
          >
            <Trash2 size={16} />
            Remove
          </button>
        </div>
        
        <div className="mipmap-list">
          {mipmapInfo.map((mipmap) => (
            <button
              key={mipmap.Level}
              className={`mipmap-item ${currentLevel === mipmap.Level ? 'active' : ''}`}
              onClick={() => onLevelChange(mipmap.Level)}
              disabled={disabled}
            >
              <span className="mipmap-level">Level {mipmap.Level}</span>
              <span className="mipmap-size">{mipmap.Width}×{mipmap.Height}</span>
              <span className="mipmap-bytes">{formatBytes(mipmap.Size)}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}
