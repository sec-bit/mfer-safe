import * as React from "react";
import { useState, useEffect } from "react";
import TextareaAutosize from "@mui/material/TextareaAutosize";
import { emit, listen } from "@tauri-apps/api/event";

const MAX_LOG_LEN = 10;
export default function LogView() {
  const [log, setLog] = useState("");
  useEffect(() => {
    listen("mfernode-event", (event) => {
      if (event.payload !== undefined) {
        setLog((log) => event.payload + "\n" + log);
      }
    });
  }, []);
  return (
    <TextareaAutosize
      aria-label="empty textarea"
      placeholder="Log..."
      style={{ width: "100%" }}
      value={log}
    />
  );
}
