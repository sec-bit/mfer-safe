import * as React from "react";
import { useEffect } from "react";
import TextareaAutosize from "@mui/material/TextareaAutosize";
import { listen } from "@tauri-apps/api/event";

export default function LogView(props) {
  const { log, setLog } = props;
  useEffect(() => {
    listen("mfernode-event", (event) => {
      if (event.payload !== undefined) {
        setLog((log) => {
          var logLines = log.split("\n").slice(0, 500);
          log = logLines.join("\n");
          return event.payload + "\n" + log
        });
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
