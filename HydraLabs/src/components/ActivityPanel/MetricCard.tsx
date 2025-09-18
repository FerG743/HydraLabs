export function MetricCard({ metric, index, getIcon }) {
  const [key, value] = metric;
  const MetricIcon = getIcon(key.toLowerCase(), "default");

  return (
    <div
      key={key}
      className="p-4 bg-gray-50 rounded-lg animate-scale-in"
      style={{ animationDelay: `${index * 150}ms` }}
    >
      <div className="flex items-center gap-2 mb-2">
        <MetricIcon className="w-5 h-5 text-blue-500" />
        <span className="text-sm text-gray-600 capitalize">
          {key.replace(/([A-Z])/g, " $1")}
        </span>
      </div>
      <div className="text-2xl font-bold text-gray-900">{String(value)}</div>
    </div>
  );
}
