interface EmptyModuleViewProps {
  title: string;
}

export default function EmptyModuleView({ title }: EmptyModuleViewProps) {
  return (
    <section className="min-h-[calc(100vh-12rem)]">
      <div className="rounded-[18px] border-2 border-dashed border-gold-champagne/25 bg-white p-10">
        <div className="flex items-center gap-3 border-l-[6px] border-gold-champagne pl-4">
          <h1 className="text-[1.45rem] font-black text-text-title">{title}</h1>
          <span className="rounded-[5px] bg-hero-bg px-3 py-1 font-mono text-[0.72rem] font-black uppercase tracking-[0.16em] text-gold-champagne">
            Empty Module
          </span>
        </div>
        <div className="mt-8 min-h-[460px] rounded-[14px] border border-gold-champagne/15 bg-hero-bg/45" />
      </div>
    </section>
  );
}
