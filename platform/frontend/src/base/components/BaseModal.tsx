import React from "react";
import { X } from "lucide-react";

export interface BaseModalProps {
  open: boolean;
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  onClose: () => void;
}

export function BaseModal({ open, title, children, footer, onClose }: BaseModalProps) {
  if (!open) return null;

  return (
    <div className="modal-backdrop fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-4">
      <section className="modal-panel w-full max-w-lg rounded-[8px] bg-white shadow-xl">
        <header className="flex items-center justify-between border-b border-slate-200 px-5 py-4">
          <h2 className="text-base font-semibold text-slate-900">{title}</h2>
          <button className="rounded p-1 text-slate-500 hover:bg-slate-100" onClick={onClose} type="button" title="关闭">
            <X className="h-4 w-4" />
          </button>
        </header>
        <div className="px-5 py-4">{children}</div>
        {footer && <footer className="flex justify-end gap-2 border-t border-slate-200 px-5 py-4">{footer}</footer>}
      </section>
    </div>
  );
}
