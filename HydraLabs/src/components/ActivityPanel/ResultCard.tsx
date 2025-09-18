export function ResultCard({ result, index, getStatusStyle, formatTime, onClick }) {
  const statusStyle = getStatusStyle(result.status);
  const StatusIcon = statusStyle.icon;

  return (
    <div
      key={result.id || index}
      className="p-4 border border-gray-200 rounded-lg hover:shadow-sm transition-all duration-200 cursor-pointer animate-fade-in-up"
      style={{ animationDelay: `${index * 100}ms` }}
      onClick={() => onClick && onClick(result, index)}
    >
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-3">
          <StatusIcon className={`w-4 h-4 ${statusStyle.color}`} />
          <h4 className="font-medium">{result.name || result.title || "Result"}</h4>
        </div>
        <span className="text-xs text-gray-500">
          {formatTime(result.timestamp)}
        </span>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
        {Object.entries(result).map(([key, value]) => {
          if (["id", "name", "title", "timestamp", "status"].includes(key)) return null;
          return (
            <div key={key}>
              <span className="text-gray-500 capitalize">{key}:</span>
              <span className="ml-1 font-medium">{String(value)}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
