import { useState } from 'react';
import { ImagePlus, Layers } from 'lucide-react';

interface CompositionPanelProps {
  disabled: boolean;
  onReplace: () => void;
  onOverlay: (opacity: number, fit: boolean) => void;
}

export function CompositionPanel({ disabled, onReplace, onOverlay }: CompositionPanelProps) {
  const [opacity, setOpacity] = useState(100);
  const [fit, setFit] = useState(true);

  return (
    <div className="panel">
      <div className="panel-header">
        <Layers size={18} />
        <h3>Composition</h3>
      </div>
      <div className="panel-content">
        <button className="action-btn" onClick={onReplace} disabled={disabled}>
          <ImagePlus size={16} />
          Replace Image
        </button>

        <div className="composition-group">
          <label>Overlay opacity: {opacity}%</label>
          <input
            type="range"
            min="0"
            max="100"
            value={opacity}
            onChange={(e) => setOpacity(Number(e.target.value))}
            disabled={disabled}
          />
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={fit}
              onChange={(e) => setFit(e.target.checked)}
              disabled={disabled}
            />
            Stretch overlay to fit
          </label>
          <button className="action-btn" onClick={() => onOverlay(opacity, fit)} disabled={disabled}>
            <ImagePlus size={16} />
            Overlay Image
          </button>
        </div>
      </div>
    </div>
  );
}
