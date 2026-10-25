import styles from "../page.module.css";
import ButtonLink from "@/components/ui/ButtonLink";

export default function LoginPage() {

  return (
    <div className={styles.page}>
      <main className={styles.main}>
        <div className={styles.intro}>
          <h1>
            <code className={styles.code}>Login</code>
          </h1>
        </div>
        
        <div className={styles.intro}>
          Username
        </div>

        <input
          type="text"
          placeholder="Input username"
          className={styles.searchInput}
        />

        <div className={styles.intro}>
          Password
        </div>

        <input
          type="text"
          placeholder="Input password"
          className={styles.searchInput}
        />

        <div className={styles.ctas}>
          <ButtonLink href="/admin" variant="ghost" size="sm" className={styles.primary}>
            Enter
          </ButtonLink>
        </div>
      </main>
    </div>
  );
}
