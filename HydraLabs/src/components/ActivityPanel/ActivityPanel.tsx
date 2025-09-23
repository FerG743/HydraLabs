import React from "react";
import { Eye, Trash2, Download } from "lucide-react";
import { DEFAULT_ICONS } from "./constants";
import { formatTime, copyToClipboard } from "./ActivityPanleUtils";
import { ActivityItem } from "./ActivityItem";
import { ResultCard } from "./ResultCard";
import { MetricCard } from "./MetricCard";
import { useActivityPanel } from "./useActivityPanel";

export default function ActivityPanel({
  activities = [],
  metrics = {},
  results = [],
  height = 300,
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
  renderHeader,
  className = "",
}) {
  const {
    activeTab,
    setActiveTab,
    filteredActivities,
    statusStyleMap,
    activityRef,
  } = useActivityPanel({ activities, autoScroll });

  const iconMap = { ...DEFAULT_ICONS };

  if (!isVisible) return null;

  return (
    <div
      className={`bg-white border-t border-gray-200 animate-slide-up ${className}`}
      style={{ height: `${height}px` }}
    >
      {/* Header */}
      {renderHeader ? renderHeader({ title, metrics, onToggle }) : (
        <div className="flex items-center justify-between p-3 border-b bg-gray-50">
          <h3 className="font-medium">{title}</h3>
          <div className="flex gap-2">
            {showClear && (
              <button onClick={onClear} className="p-1 hover:text-red-600">
                <Trash2 className="w-4 h-4" />
              </button>
            )}
            {showExport && (
              <button onClick={onExport} className="p-1 hover:text-blue-600">
                <Download className="w-4 h-4" />
              </button>
            )}
            {onToggle && (
              <button onClick={onToggle} className="p-1 hover:text-gray-700">
                <Eye className="w-4 h-4" />
              </button>
            )}
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="flex border-b">
        {["activity", "results", "metrics"].map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm ${
              activeTab === tab
                ? "bg-blue-50 text-blue-700 border-b-2 border-blue-500"
                : "text-gray-600 hover:text-gray-800"
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="h-full overflow-hidden">
        {activeTab === "activity" && (
          <div ref={activityRef} className="flex-1 overflow-y-auto p-2 space-y-1">
            {filteredActivities.map((a, i) => (
              <ActivityItem
                key={a.id || i}
                activity={a}
                index={i}
                getStatusStyle={(s) => statusStyleMap[s] || statusStyleMap.info}
                getIcon={(t) => iconMap[t] || iconMap.default}
                copyToClipboard={copyToClipboard}
                onClick={onActivityClick}
              />
            ))}
          </div>
        )}

        {activeTab === "results" && (
          <div className="p-4 overflow-y-auto h-full space-y-3">
            {results.map((r, i) => (
              <ResultCard
                key={r.id || i}
                result={r}
                index={i}
                getStatusStyle={(s) => statusStyleMap[s] || statusStyleMap.info}
                formatTime={formatTime}
                onClick={onResultClick}
              />
            ))}
          </div>
        )}

        {activeTab === "metrics" && (
          <div className="p-4 overflow-y-auto h-full grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
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
  );
}
