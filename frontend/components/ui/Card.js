export default function Card({ title, children, footer }) {
  return (
    <section className="card">
      {title && <h3 className="card-title">{title}</h3>}
      {children}
      {footer && <div className="card-footer">{footer}</div>}
    </section>
  );
}