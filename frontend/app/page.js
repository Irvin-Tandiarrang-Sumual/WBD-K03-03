import Image from "next/image";
import styles from "./page.module.css";
// import { Card } from "next/font/google";
import Card from "@/components/ui/Card";
import ButtonLink from "@/components/ui/ButtonLink"

const features = [
  {title: "Databases", text: "View databases, tables. Compare versions. Migrate to different versions.", href: "/databases"},
  {title: "Admin Control", text: "Create users. Create and edit access groups. Handle sessions. ", href: "/admin"}
]

export default function Home() {

  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <div className={styles.intro}>
          <h1>
            <code className={styles.code}>Wedebe</code>
          </h1>
          <p>
            A minimalist Database Management Platform, created with connection to PostgreSQL. 
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

        <div className={styles.ctas}>
          <ButtonLink href = "/login" variant="ghost" size = "sm" className = {styles.primary}>
            Login
          </ButtonLink>
        </div>
      </main>
    </div>
  );
}
