import { useState, useEffect, useRef } from "react";
import { STATUS_STYLES } from "./constants";

export function useActivityPanel({
  activities,
  defaultTab = "activity",
  defaultFilter = "all",
  autoScroll = true,
  statusStyles = {},
}) {
  const [activeTab, setActiveTab] = useState(defaultTab);
  const [filter, setFilter] = useState(defaultFilter);
  const [internalAutoScroll, setInternalAutoScroll] = useState(autoScroll);
  const activityRef = useRef<HTMLDivElement>(null);

  // Merge custom styles with defaults
  const statusStyleMap = { ...STATUS_STYLES, ...statusStyles };

  // Filter activities
  const filteredActivities = activities.filter((a) => {
    if (filter === "all") return true;
    return a.status === filter || a.type === filter;
  });

  // Handle auto-scroll
  useEffect(() => {
    if (internalAutoScroll && activityRef.current) {
      activityRef.current.scrollTop = activityRef.current.scrollHeight;
    }
  }, [activities, internalAutoScroll]);

  return {
    activeTab,
    setActiveTab,
    filter,
    setFilter,
    internalAutoScroll,
    setInternalAutoScroll,
    filteredActivities,
    statusStyleMap,
    activityRef,
  };
}
