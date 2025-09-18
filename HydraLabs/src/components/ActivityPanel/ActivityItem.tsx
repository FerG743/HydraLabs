import { Copy } from "lucide-react";

export function ActivityItem({
  activity,
  index,
  getStatusStyle,
  getIcon,
  onClick,
  copyToClipboard,
}) {
  const statusStyle = getStatusStyle(activity.status);
  const TypeIcon = getIcon(activity.type);
  const StatusIcon = statusStyle.icon;

  return (
    <div
      key={activity.id || index}
      className="flex items-start gap-3 p-2 hover:bg-gray-50 rounded group cursor-pointer animate-fade-in-up"
      style={{ animationDelay: `${index * 50}ms` }}
      onClick={() => onClick && onClick(activity, index)}
    >
      <div className="flex items-center gap-2 min-w-0">
        <StatusIcon className={`w-4 h-4 ${statusStyle.color}`} />
        <TypeIcon className="w-4 h-4 text-gray-500" />
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between">
          <p className="text-sm font-medium text-gray-900 truncate">
            {activity.message || activity.title || "No message"}
          </p>
          <span className="text-xs text-gray-500 ml-2">
            {activity.timestamp}
          </span>
        </div>
      </div>

      <button
        onClick={(e) => {
          e.stopPropagation();
          copyToClipboard(activity.message || JSON.stringify(activity));
        }}
        className="opacity-0 group-hover:opacity-100 p-1 text-gray-400 hover:text-gray-600 transition-all duration-200"
        title="Copy"
      >
        <Copy className="w-3 h-3" />
      </button>
    </div>
  );
}
