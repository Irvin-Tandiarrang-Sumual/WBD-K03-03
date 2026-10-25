import styles from "./AdminDashboard.module.css";
import Card from "@/components/ui/Card";
import ButtonLink from "@/components/ui/ButtonLink"

const features = [
  {title: "Connections", text: "View connections", href: "/admin/connections"},
  {title: "Groups", text: "Create and edit access groups", href: "/admin/groups"},
  {title: "Users", text: "Create and edit users", href: "/admin/users"},
]

export default function AdminDashboard() {

  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <div className={styles.intro}>
          <h1>
            <code className={styles.code}>Admin Control</code>
          </h1>
          <p>
            Admin controls ALL!!!
          </p>
        </div>
        
        <div className={styles.grid}>
          {features.map((f) => (
            <Card
              key={f.href}
              title={
                <ButtonLink href = {f.href} variant="ghost" size = "sm">
                  {f.title}
                </ButtonLink>
              }
            >
              <p className="muted">{f.text}</p>
            </Card>
          ))}
        </div>
      </main>
    </div>
  );
}
