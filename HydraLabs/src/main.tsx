import React from "react";
import ReactDOM from "react-dom/client";
import APITesting from "./pages/APITesting";
import "./App.css";

// Force CSS variables to be available
document.documentElement.style.setProperty('--card', '315 20% 9%');
document.documentElement.style.setProperty('--card-foreground', '60 9% 98%');
document.documentElement.style.setProperty('--border', '315 20% 18%');

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <APITesting />
  </React.StrictMode>
);