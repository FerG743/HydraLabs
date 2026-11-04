import React, { useRef, useState, useEffect } from "react";
import { Eye, Trash2, Download } from "lucide-react";
import { DEFAULT_ICONS } from "./hooks/Constants";
import { formatTime, copyToClipboard } from "./utils/ActivityPanleUtils";
import { ActivityItem, ResultCard, MetricCard } from "./components";
import { useActivityPanel } from "./hooks";
import { useResizer } from "@/hooks";
import { ResizeHandle } from "./components";

export default function ActivityPanel({
  activities = [],
  metrics = {},
  results = [],
  height: initialHeight = 300,
  minHeight = 150,
  maxHeight = 600,
  isVisible = true,
  title = "Activity Panel",
  autoScroll = true,
  showClear = true,
  showExport = true,
  onToggle,
  onExport,
  onClear,
  onActivityClick,
  onResultClick,
  onHeightChange,
  renderHeader,
  className = "",
} = {}) {
  const {
    activeTab,
    setActiveTab,
    filteredActivities,
    statusStyleMap,
    activityRef,
  } = useActivityPanel({ activities, autoScroll });

  // Add resizer hook
  const { width: height, isResizing, handleMouseDown } = useResizer({
    initialWidth: initialHeight,
    minWidth: minHeight,
    maxWidth: maxHeight,
    onWidthChange: onHeightChange,
    direction: 'vertical'
  });

  const iconMap = { ...DEFAULT_ICONS };

  const scrollRef = useRef(null);
  const [showRightFade, setShowRightFade] = useState(false);

  useEffect(() => {
    const updateFade = () => {
      if (scrollRef.current) {
        setShowRightFade(
          scrollRef.current.scrollWidth > scrollRef.current.clientWidth
        );
      }
    };

    updateFade();
    window.addEventListener("resize", updateFade);
    return () => window.removeEventListener("resize", updateFade);
  }, [activeTab, filteredActivities, results, metrics]);

  if (!isVisible) return null;

  return (
    <div className="flex flex-col">
      {/* Resize Handle at the top */}
      <ResizeHandle
        isResizing={isResizing}
        onMouseDown={handleMouseDown}
        direction="vertical"
      />
      
      <div
        className={`flex flex-col bg-card text-card-foreground border-t border-border animate-slide-up ${className}`}
        style={{ height: `${height}px` }}
      >
        {/* Header */}
        {renderHeader ? (
          renderHeader({ title, metrics, onToggle })
        ) : (
          <div className="flex items-center justify-between p-3 border-b border-border bg-card">
            <h3 className="font-medium text-foreground">{title}</h3>
            <div className="flex gap-2">
              {showClear && (
                <button
                  onClick={onClear}
                  className="p-1 text-muted-foreground hover:text-destructive transition-colors"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              )}
              {showExport && (
                <button
                  onClick={onExport}
                  className="p-1 text-muted-foreground hover:text-primary transition-colors"
                >
                  <Download className="w-4 h-4" />
                </button>
              )}
              {onToggle && (
                <button
                  onClick={onToggle}
                  className="p-1 text-muted-foreground hover:text-foreground transition-colors"
                >
                  <Eye className="w-4 h-4" />
                </button>
              )}
            </div>
          </div>
        )}

        {/* Tabs */}
        <div className="flex border-b border-border">
          {["activity", "results", "metrics"].map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 text-sm transition-colors ${
                activeTab === tab
                  ? "bg-primary/10 text-primary border-b-2 border-primary"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {tab}
            </button>
          ))}
        </div>

        {/* Scrollable Content with dynamic fade */}
        <div className="relative flex-1 overflow-hidden bg-card">
          {showRightFade && (
            <div className="pointer-events-none absolute top-0 right-0 h-full w-6 bg-gradient-to-l from-card to-transparent z-10" />
          )}

          <div
            ref={scrollRef}
            className="h-full overflow-x-auto overflow-y-auto p-2 pr-4"
          >
            {activeTab === "activity" && (
              <div ref={activityRef} className="space-y-1">
                {filteredActivities.map((a, i) => (
                  <ActivityItem
                    key={a.id || i}
                    activity={a}
                    index={i}
                    getStatusStyle={(s) =>
                      statusStyleMap[s] || statusStyleMap.info
                    }
                    getIcon={(t) => iconMap[t] || iconMap.default}
                    copyToClipboard={copyToClipboard}
                    onClick={onActivityClick}
                  />
                ))}
              </div>
            )}

            {activeTab === "results" && (
              <div className="space-y-3">
                {results.map((r, i) => (
                  <ResultCard
                    key={r.id || i}
                    result={r}
                    index={i}
                    getStatusStyle={(s) =>
                      statusStyleMap[s] || statusStyleMap.info
                    }
                    formatTime={formatTime}
                    onClick={onResultClick}
                  />
                ))}
              </div>
            )}

            {activeTab === "metrics" && (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {Object.entries(metrics).map((m, i) => (
                  <MetricCard
                    key={m[0]}
                    metric={m}
                    index={i}
                    getIcon={(t) => iconMap[t] || iconMap.default}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}