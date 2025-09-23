import {
  Activity, Globe, Target, Shield, Zap, Bug, Info,
  Users, Settings, Database, CheckCircle, XCircle,
  AlertCircle, Clock
} from "lucide-react";

export const DEFAULT_ICONS = {
  request: Globe,
  test: Target,
  validation: Shield,
  performance: Zap,
  error: Bug,
  info: Info,
  user: Users,
  system: Settings,
  api: Database,
  security: Shield,
  code: Activity,
  default: Activity,
};

export const STATUS_STYLES = {
  success: { icon: CheckCircle, color: "text-green-500", bg: "bg-green-100", text: "text-green-700" },
  passed:  { icon: CheckCircle, color: "text-green-500", bg: "bg-green-100", text: "text-green-700" },
  error:   { icon: XCircle,     color: "text-red-500",   bg: "bg-red-100",   text: "text-red-700" },
  failed:  { icon: XCircle,     color: "text-red-500",   bg: "bg-red-100",   text: "text-red-700" },
  warning: { icon: AlertCircle, color: "text-yellow-500",bg: "bg-yellow-100",text: "text-yellow-700" },
  info:    { icon: Info,        color: "text-blue-500",  bg: "bg-blue-100",  text: "text-blue-700" },
  pending: { icon: Clock,       color: "text-gray-500",  bg: "bg-gray-100",  text: "text-gray-700" },
};
