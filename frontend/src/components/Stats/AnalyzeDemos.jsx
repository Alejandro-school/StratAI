// frontend/src/components/Stats/AnalyzeDemos.jsx
import React, { useState, useCallback } from "react";
import SectionHeader from "./common/SectionHeader";
import Card from "./common/Card";
import EmptyState from "./common/EmptyState";
import ErrorState from "./common/ErrorState";
import { Upload, FileText, CheckCircle, XCircle, Clock, Eye, Trash2 } from "lucide-react";
import "../../styles/Stats/analyzeDemos.css";

/**
 * MOCK DATA - Ejemplo de estructura de datos esperada
 */
const MOCK_DATA = {
  queue: [
    { 
      id: "demo_1", 
      filename: "match_mirage_2025-10-14.dem", 
      size: "45.2 MB",
      status: "completed", 
      uploadDate: "2025-10-14 15:30",
      processedDate: "2025-10-14 15:35",
      map: "Mirage",
      rounds: 24
    },
    { 
      id: "demo_2", 
      filename: "match_inferno_2025-10-13.dem", 
      size: "38.7 MB",
      status: "processing", 
      uploadDate: "2025-10-13 18:20",
      progress: 65
    },
    { 
      id: "demo_3", 
      filename: "match_dust2_2025-10-12.dem", 
      size: "52.1 MB",
      status: "queued", 
      uploadDate: "2025-10-12 20:10"
    },
    { 
      id: "demo_4", 
      filename: "match_nuke_2025-10-11.dem", 
      size: "41.3 MB",
      status: "error", 
      uploadDate: "2025-10-11 22:45",
      error: "Error al parsear demo: archivo corrupto"
    }
  ]
};

/**
 * AnalyzeDemos - Componente para subir y procesar demos
 * 
 * @param {Object} data - Datos de demos { queue }
 * @param {boolean} loading - Estado de carga
 * @param {string} error - Mensaje de error si existe
 * @param {Function} onRetry - Función para reintentar carga
 * @param {Function} onProcess - Función para iniciar procesamiento
 * @param {Function} onViewResults - Función para ver resultados de demo
 * @param {Function} onUpload - Función para subir nueva demo
 * @param {Function} onDelete - Función para eliminar demo
 */
export default function AnalyzeDemos({ 
  data, 
  loading = false, 
  error = null, 
  onRetry,
  onProcess,
  onViewResults,
  onUpload,
  onDelete 
}) {
  const [dragActive, setDragActive] = useState(false);
  
  // Usar mock data si no se proveen datos
  const statsData = data || MOCK_DATA;
  const { queue } = statsData;

  // Handler para procesar archivos
  const handleFiles = useCallback((files) => {
    if (onUpload) {
      onUpload(files);
    } else {
      console.log("Archivos seleccionados:", files);
    }
  }, [onUpload]);

  // Drag & Drop handlers (deben estar antes de los returns condicionales)
  const handleDrag = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      const files = Array.from(e.dataTransfer.files);
      handleFiles(files);
    }
  }, [handleFiles]);

  const handleFileInput = useCallback((e) => {
    if (e.target.files && e.target.files[0]) {
      const files = Array.from(e.target.files);
      handleFiles(files);
    }
  }, [handleFiles]);

  // Estado de carga
  if (loading) {
    return (
      <div className="ad-container">
        <SectionHeader 
          title="Analizar demos" 
          description="Sube demos para su análisis detallado"
        />
        <Card>
          <div className="ad-loading">
            <div className="ad-spinner"></div>
            <p>Cargando demos...</p>
          </div>
        </Card>
      </div>
    );
  }

  // Estado de error
  if (error) {
    return (
      <div className="ad-container">
        <SectionHeader 
          title="Analizar demos" 
          description="Sube demos para su análisis detallado"
        />
        <Card>
          <ErrorState message={error} onRetry={onRetry} />
        </Card>
      </div>
    );
  }

  // Status icon y color
  const getStatusIcon = (status) => {
    switch (status) {
      case "completed": return <CheckCircle size={20} />;
      case "processing": return <Clock size={20} />;
      case "queued": return <Clock size={20} />;
      case "error": return <XCircle size={20} />;
      default: return <FileText size={20} />;
    }
  };

  const getStatusClass = (status) => {
    switch (status) {
      case "completed": return "ad-status-completed";
      case "processing": return "ad-status-processing";
      case "queued": return "ad-status-queued";
      case "error": return "ad-status-error";
      default: return "ad-status-default";
    }
  };

  const getStatusLabel = (status) => {
    switch (status) {
      case "completed": return "Completado";
      case "processing": return "Procesando";
      case "queued": return "En cola";
      case "error": return "Error";
      default: return "Desconocido";
    }
  };

  return (
    <div className="ad-container">
      <SectionHeader 
        title="Analizar demos" 
        description="Sube demos para su análisis detallado"
      />

      {/* Upload Zone */}
      <Card className="ad-upload-card">
        <div 
          className={`ad-dropzone ${dragActive ? 'ad-dropzone-active' : ''}`}
          onDragEnter={handleDrag}
          onDragLeave={handleDrag}
          onDragOver={handleDrag}
          onDrop={handleDrop}
        >
          <input 
            type="file" 
            id="ad-file-input" 
            className="ad-file-input"
            accept=".dem"
            multiple
            onChange={handleFileInput}
          />
          <label htmlFor="ad-file-input" className="ad-dropzone-content">
            <div className="ad-upload-icon">
              <Upload size={48} />
            </div>
            <h3 className="ad-upload-title">
              Arrastra demos aquí o haz click para seleccionar
            </h3>
            <p className="ad-upload-subtitle">
              Soporta archivos .dem de CS2. Puedes subir múltiples archivos a la vez.
            </p>
            <button className="ad-upload-btn">
              <Upload size={18} />
              <span>Seleccionar archivos</span>
            </button>
          </label>
        </div>
      </Card>

      {/* Queue List */}
      <Card className="ad-queue-card">
        <div className="ad-queue-header">
          <div>
            <h3 className="ad-queue-title">Demos subidas</h3>
            <p className="ad-queue-subtitle">
              {queue && queue.length > 0 
                ? `${queue.length} demo${queue.length > 1 ? 's' : ''} en total`
                : 'No hay demos aún'
              }
            </p>
          </div>
        </div>

        {queue && queue.length > 0 ? (
          <div className="ad-queue-list">
            {queue.map((demo) => (
              <div key={demo.id} className="ad-demo-item">
                <div className={`ad-demo-status ${getStatusClass(demo.status)}`}>
                  {getStatusIcon(demo.status)}
                </div>
                
                <div className="ad-demo-content">
                  <div className="ad-demo-header-row">
                    <span className="ad-demo-filename">{demo.filename}</span>
                    <span className="ad-demo-size">{demo.size}</span>
                  </div>
                  
                  <div className="ad-demo-info">
                    <span className={`ad-demo-badge ${getStatusClass(demo.status)}`}>
                      {getStatusLabel(demo.status)}
                    </span>
                    <span className="ad-demo-date">Subida: {demo.uploadDate}</span>
                    {demo.map && <span className="ad-demo-map">Mapa: {demo.map}</span>}
                    {demo.rounds && <span className="ad-demo-rounds">{demo.rounds} rondas</span>}
                  </div>

                  {demo.status === "processing" && demo.progress && (
                    <div className="ad-progress-bar">
                      <div 
                        className="ad-progress-fill" 
                        style={{ width: `${demo.progress}%` }}
                      />
                      <span className="ad-progress-text">{demo.progress}%</span>
                    </div>
                  )}

                  {demo.status === "error" && demo.error && (
                    <div className="ad-error-message">
                      <XCircle size={14} />
                      <span>{demo.error}</span>
                    </div>
                  )}
                </div>

                <div className="ad-demo-actions">
                  {demo.status === "completed" && (
                    <button 
                      className="ad-action-btn ad-btn-view"
                      onClick={() => onViewResults && onViewResults(demo.id)}
                      title="Ver resultados"
                    >
                      <Eye size={18} />
                    </button>
                  )}
                  {demo.status === "queued" && (
                    <button 
                      className="ad-action-btn ad-btn-process"
                      onClick={() => onProcess && onProcess(demo.id)}
                      title="Procesar ahora"
                    >
                      <Clock size={18} />
                    </button>
                  )}
                  {(demo.status === "error" || demo.status === "completed") && (
                    <button 
                      className="ad-action-btn ad-btn-delete"
                      onClick={() => onDelete && onDelete(demo.id)}
                      title="Eliminar"
                    >
                      <Trash2 size={18} />
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <EmptyState 
            icon={FileText}
            title="No hay demos"
            description="Sube tus primeras demos para comenzar el análisis"
          />
        )}
      </Card>
    </div>
  );
}

/**
 * EJEMPLO DE USO EN ROUTER:
 * 
 * import AnalyzeDemos from './components/Stats/AnalyzeDemos';
 * 
 * const demosData = {
 *   queue: [...]
 * };
 * 
 * const handleUpload = (files) => {
 *   console.log('Subiendo:', files);
 * };
 * 
 * const handleProcess = (demoId) => {
 *   console.log('Procesando:', demoId);
 * };
 * 
 * const handleViewResults = (demoId) => {
 *   console.log('Ver resultados:', demoId);
 * };
 * 
 * <AnalyzeDemos 
 *   data={demosData} 
 *   loading={false}
 *   error={null}
 *   onRetry={() => fetchData()}
 *   onUpload={handleUpload}
 *   onProcess={handleProcess}
 *   onViewResults={handleViewResults}
 *   onDelete={(id) => console.log('Delete:', id)}
 * />
 */




