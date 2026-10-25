import { Layers } from 'lucide-react';

interface FormatSelectorProps {
  currentFormat: number;
  formats: any[];
  onConvert: (format: number) => void;
  disabled: boolean;
}

export function FormatSelector({ currentFormat, formats, onConvert, disabled }: FormatSelectorProps) {
  return (
    <div className="panel">
      <div className="panel-header">
        <Layers size={18} />
        <h3>Format</h3>
      </div>
      <div className="panel-content">
        <div className="format-grid">
          {formats.map((format) => (
            <button
              key={format.id}
              className={`format-btn ${currentFormat === format.id ? 'active' : ''}`}
              onClick={() => onConvert(format.id)}
              disabled={disabled}
              title={format.description}
            >
              {format.name}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
