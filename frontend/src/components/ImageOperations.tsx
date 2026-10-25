import { Sliders } from 'lucide-react';
import { useState } from 'react';

interface ImageOperationsProps {
  onOperation: (operation: string, params: any) => void;
  disabled: boolean;
}

export function ImageOperations({ onOperation, disabled }: ImageOperationsProps) {
  const [brightness, setBrightness] = useState(100);
  const [contrast, setContrast] = useState(100);
  const [saturation, setSaturation] = useState(100);

  const handleBrightness = () => {
    onOperation('brightness', { factor: brightness / 100 });
    setBrightness(100);
  };

  const handleContrast = () => {
    onOperation('contrast', { factor: contrast / 100 });
    setContrast(100);
  };

  const handleSaturation = () => {
    onOperation('saturation', { factor: saturation / 100 });
    setSaturation(100);
  };

  const handleInvert = () => {
    onOperation('invert', {});
  };

  const handleGrayscale = () => {
    onOperation('grayscale', {});
  };

  const handleFlipHorizontal = () => {
    onOperation('flip-horizontal', {});
  };

  const handleFlipVertical = () => {
    onOperation('flip-vertical', {});
  };

  return (
    <div className="panel">
      <div className="panel-header">
        <Sliders size={18} />
        <h3>Adjustments</h3>
      </div>
      <div className="panel-content">
        <div className="adjustment-group">
          <label>Brightness: {brightness}%</label>
          <input
            type="range"
            min="0"
            max="200"
            value={brightness}
            onChange={(e) => setBrightness(Number(e.target.value))}
            disabled={disabled}
          />
          <button onClick={handleBrightness} disabled={disabled} className="small-btn">
            Apply
          </button>
        </div>

        <div className="adjustment-group">
          <label>Contrast: {contrast}%</label>
          <input
            type="range"
            min="0"
            max="200"
            value={contrast}
            onChange={(e) => setContrast(Number(e.target.value))}
            disabled={disabled}
          />
          <button onClick={handleContrast} disabled={disabled} className="small-btn">
            Apply
          </button>
        </div>

        <div className="adjustment-group">
          <label>Saturation: {saturation}%</label>
          <input
            type="range"
            min="0"
            max="200"
            value={saturation}
            onChange={(e) => setSaturation(Number(e.target.value))}
            disabled={disabled}
          />
          <button onClick={handleSaturation} disabled={disabled} className="small-btn">
            Apply
          </button>
        </div>

        <div className="operation-buttons">
          <button onClick={handleInvert} disabled={disabled} className="op-btn">
            Invert
          </button>
          <button onClick={handleGrayscale} disabled={disabled} className="op-btn">
            Grayscale
          </button>
          <button onClick={handleFlipHorizontal} disabled={disabled} className="op-btn">
            Flip H
          </button>
          <button onClick={handleFlipVertical} disabled={disabled} className="op-btn">
            Flip V
          </button>
        </div>
      </div>
    </div>
  );
}
