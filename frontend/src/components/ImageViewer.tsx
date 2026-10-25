import { useState, useRef, useEffect } from 'react';
import { ZoomIn, ZoomOut, RotateCw, Maximize2 } from 'lucide-react';

interface ImageViewerProps {
  image: string;
  fileInfo: any;
  loading: boolean;
}

export function ImageViewer({ image, fileInfo, loading }: ImageViewerProps) {
  const [zoom, setZoom] = useState(1);
  const [rotation, setRotation] = useState(0);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const containerRef = useRef<HTMLDivElement>(null);

  const handleZoomIn = () => {
    setZoom(prev => Math.min(prev * 1.2, 10));
  };

  const handleZoomOut = () => {
    setZoom(prev => Math.max(prev / 1.2, 0.1));
  };

  const handleResetZoom = () => {
    setZoom(1);
    setRotation(0);
    setPan({ x: 0, y: 0 });
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button === 0) {
      setIsDragging(true);
      setDragStart({ x: e.clientX - pan.x, y: e.clientY - pan.y });
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isDragging) {
      setPan({
        x: e.clientX - dragStart.x,
        y: e.clientY - dragStart.y
      });
    }
  };

  const handleMouseUp = () => {
    setIsDragging(false);
  };

  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setZoom(prev => Math.max(0.1, Math.min(prev * delta, 10)));
  };

  return (
    <div className="image-viewer">
      <div className="viewer-toolbar">
        <button onClick={handleZoomOut} title="Zoom Out">
          <ZoomOut size={18} />
        </button>
        <span className="zoom-level">{Math.round(zoom * 100)}%</span>
        <button onClick={handleZoomIn} title="Zoom In">
          <ZoomIn size={18} />
        </button>
        <button onClick={handleResetZoom} title="Reset View">
          <Maximize2 size={18} />
        </button>
      </div>
      
      <div 
        ref={containerRef}
        className="viewer-container"
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onWheel={handleWheel}
      >
        {loading ? (
          <div className="loading-overlay">
            <div className="spinner"></div>
            <p>Loading...</p>
          </div>
        ) : image ? (
          <img
            src={image}
            alt="BLP Preview"
            className="viewer-image"
            style={{
              transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom}) rotate(${rotation}deg)`,
              cursor: isDragging ? 'grabbing' : 'grab'
            }}
            draggable={false}
          />
        ) : (
          <div className="no-image">
            <p>No image loaded</p>
          </div>
        )}
      </div>
      
      {fileInfo && (
        <div className="image-info">
          <span>{fileInfo.width} × {fileInfo.height}</span>
          <span>{fileInfo.formatName}</span>
          {fileInfo.hasAlpha && <span className="alpha-badge">Alpha</span>}
        </div>
      )}
    </div>
  );
}
