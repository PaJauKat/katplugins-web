"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { PluginAccess } from "@/lib/types";
import PluginCard from "./PluginCard";

export default function PluginsGrid() {
  const [plugins, setPlugins] = useState<PluginAccess[] | null>(null);

  useEffect(() => {
    api
      .plugins()
      .then((res) => setPlugins(res.plugins))
      .catch(() => setPlugins([]));
  }, []);

  if (plugins === null) {
    return (
      <div className="mt-8 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="card h-40 animate-pulse" />
        ))}
      </div>
    );
  }

  if (plugins.length === 0) {
    return (
      <div className="card mt-8 p-8 text-center text-slate-400">
        Aun no hay plugins publicados.
      </div>
    );
  }

  return (
    <div className="mt-8 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
      {plugins.map((pa) => (
        <PluginCard key={pa.plugin.id} plugin={pa.plugin} />
      ))}
    </div>
  );
}
