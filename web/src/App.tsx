import {
    ArrowRight,
    Bell,
    Check,
    CircleAlert,
    Files,
    Grid2x2,
    List,
    LockKeyhole,
    LogOut,
    Megaphone,
    Menu,
    Settings,
    ShieldCheck,
    Sparkles,
    UserRound,
} from "lucide-react";
import { FormEvent, useEffect, useState } from "react";
import {
    Link,
    Navigate,
    NavLink,
    Route,
    Routes,
    useNavigate,
    useParams,
} from "react-router-dom";
import { isMfaRequiredError } from "./auth-flow";
import { Button, Field, Input, Logo } from "./components/ui";
import {
    api,
    type Campaign,
    type Contact,
    type Identity,
    type Member,
    type Session,
    type Template,
} from "./lib/api";

function useSession() {
  const [session, setSession] = useState<Session | null>(() => {
    const stored = localStorage.getItem("notifiq.session");
    return stored ? JSON.parse(stored) : null;
  });
  const save = (next: Session) => {
    localStorage.setItem("notifiq.session", JSON.stringify(next));
    setSession(next);
  };
  const clear = () => {
    localStorage.removeItem("notifiq.session");
    setSession(null);
  };
  return { session, save, clear };
}

function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <main className="auth-layout">
      <section className="auth-visual">
        <Logo />
        <div className="visual-copy">
          <p className="eyebrow">
            <Sparkles size={15} /> Comunicação com intenção
          </p>
          <h1>
            Envie a mensagem certa. <em>No ritmo certo.</em>
          </h1>
          <p>
            Automatize seus disparos e transforme cada contato em uma conversa
            que importa.
          </p>
        </div>
        <div className="signal-card">
          <span className="signal-dot" />
          <span>Seu próximo envio está pronto para decolar</span>
          <ArrowRight size={18} />
        </div>
      </section>
      <section className="auth-panel">{children}</section>
    </main>
  );
}

function MfaSetupPage({
  session,
  onComplete,
}: {
  session: Session;
  onComplete: () => void;
}) {
  const [qrUrl, setQrUrl] = useState("");
  const [secret, setSecret] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    void (async () => {
      try {
        const setup = await api.setupMFA(session.access_token);
        setQrUrl(setup.otpauth_url);
        setSecret(setup.secret);
        setReady(true);
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : "Não foi possível gerar o QR Code do MFA.",
        );
      }
    })();
  }, [session.access_token]);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setLoading(true);

    try {
      await api.confirmMFA(session.access_token, code);
      onComplete();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Código MFA inválido.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthLayout>
      <div className="auth-form">
        <div className="mobile-logo">
          <Logo />
        </div>
        <p className="eyebrow">Proteja sua conta</p>
        <h2>Ative seu código MFA</h2>
        <p className="muted">
          Escaneie o QR Code abaixo no seu aplicativo autenticador e confirme o
          código gerado.
        </p>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        {ready && (
          <>
            <div className="mfa-setup-box">
              <img
                alt="QR Code do MFA"
                src={`https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=${encodeURIComponent(qrUrl)}`}
              />
              <p className="form-hint">Secret: {secret}</p>
            </div>
            <form onSubmit={submit}>
              <Field label="Código do aplicativo">
                <Input
                  required
                  inputMode="numeric"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  placeholder="000000"
                />
              </Field>
              <Button disabled={loading || code.length < 6}>
                {loading ? "Validando..." : "Ativar MFA"}
                <ShieldCheck size={18} />
              </Button>
            </form>
          </>
        )}
      </div>
    </AuthLayout>
  );
}

function AuthForm({
  mode,
  onAuth,
}: {
  mode: "login" | "register";
  onAuth: (session: Session) => void;
}) {
  const navigate = useNavigate();
  const [values, setValues] = useState({
    name: "",
    email: "",
    password: "",
    mfa_code: "",
  });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [mfaStep, setMfaStep] = useState(false);
  const isRegister = mode === "register";

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setLoading(true);

    try {
      if (isRegister) {
        const session = await api.register(
          values.name,
          values.email,
          values.password,
        );
        onAuth(session);
        navigate("/app/mfa-setup");
        return;
      }

      const session = await api.login(
        values.email,
        values.password,
        mfaStep ? values.mfa_code || undefined : undefined,
      );

      onAuth(session);
      navigate("/app");
    } catch (err) {
      if (!isRegister && isMfaRequiredError(err)) {
        setMfaStep(true);
        setError("");
        return;
      }
      setError(err instanceof Error ? err.message : "Não foi possível entrar.");
    } finally {
      setLoading(false);
    }
  };

  const resetMfaStep = () => {
    setMfaStep(false);
    setValues((current) => ({ ...current, mfa_code: "" }));
    setError("");
  };

  return (
    <AuthLayout>
      <div className="auth-form">
        <div className="mobile-logo">
          <Logo />
        </div>
        <p className="eyebrow">
          {isRegister
            ? "Comece em poucos passos"
            : mfaStep
              ? "Confirme sua identidade"
              : "Bem-vindo de volta"}
        </p>
        <h2>
          {isRegister
            ? "Crie seu espaço de envio"
            : mfaStep
              ? "Digite o código MFA"
              : "Entre na sua conta"}
        </h2>
        <p className="muted">
          {isRegister
            ? "Uma base mais inteligente para suas mensagens."
            : mfaStep
              ? "Seu acesso exige confirmação por código de autenticação."
              : "Continue de onde sua comunicação parou."}
        </p>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        <form onSubmit={submit}>
          {isRegister && (
            <Field label="Seu nome">
              <Input
                required
                value={values.name}
                onChange={(e) => setValues({ ...values, name: e.target.value })}
                placeholder="Ana Souza"
              />
            </Field>
          )}{" "}
          {!mfaStep && (
            <>
              <Field label="E-mail">
                <Input
                  required
                  type="email"
                  value={values.email}
                  onChange={(e) =>
                    setValues({ ...values, email: e.target.value })
                  }
                  placeholder="voce@empresa.com"
                />
              </Field>
              <Field label="Senha">
                <Input
                  required
                  minLength={8}
                  type="password"
                  value={values.password}
                  onChange={(e) =>
                    setValues({ ...values, password: e.target.value })
                  }
                  placeholder="Mínimo de 8 caracteres"
                />
              </Field>
            </>
          )}
          {mfaStep && (
            <Field label="Código MFA">
              <Input
                required
                inputMode="numeric"
                value={values.mfa_code}
                onChange={(e) =>
                  setValues({ ...values, mfa_code: e.target.value })
                }
                placeholder="000000"
              />
            </Field>
          )}
          <Button disabled={loading}>
            {loading
              ? "Aguarde..."
              : isRegister
                ? "Criar minha conta"
                : mfaStep
                  ? "Confirmar código"
                  : "Entrar"}
            <ArrowRight size={18} />
          </Button>
          {!isRegister && mfaStep && (
            <Button
              type="button"
              className="button-ghost"
              onClick={resetMfaStep}
              disabled={loading}
            >
              Voltar
            </Button>
          )}
        </form>
        <p className="switch-auth">
          {isRegister
            ? "Já possui uma conta? "
            : "Ainda não possui uma conta? "}
          <Link to={isRegister ? "/login" : "/register"}>
            {isRegister ? "Entrar" : "Criar conta"}
          </Link>
        </p>
        <p className="secure-note">
          <LockKeyhole size={14} /> Seus dados são protegidos por criptografia.
        </p>
      </div>
    </AuthLayout>
  );
}

function PasswordResetRequestPage() {
  const [email, setEmail] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      await api.requestPasswordReset(email);
      setMessage(
        "Se o endereço estiver cadastrado, você receberá um link de recuperação.",
      );
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível processar a solicitação.",
      );
    } finally {
      setLoading(false);
    }
  };
  return (
    <AuthLayout>
      <div className="auth-form">
        <p className="eyebrow">Acesso seguro</p>
        <h2>Recupere sua senha</h2>
        <p className="muted">
          Informe seu e-mail e enviaremos as instruções, caso exista uma conta
          correspondente.
        </p>
        {message && (
          <div className="success-box">
            <Check size={17} />
            {message}
          </div>
        )}
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        <form onSubmit={submit}>
          <Field label="E-mail">
            <Input
              required
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="voce@empresa.com"
            />
          </Field>
          <Button disabled={loading}>
            {loading ? "Enviando..." : "Enviar instruções"}
            <ArrowRight size={18} />
          </Button>
        </form>
        <p className="switch-auth">
          <Link to="/login">Voltar para o login</Link>
        </p>
      </div>
    </AuthLayout>
  );
}

function ResetPasswordPage() {
  const token = new URLSearchParams(window.location.search).get("token") ?? "";
  const navigate = useNavigate();
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (password !== confirmation) {
      setError("As senhas não conferem.");
      return;
    }
    setLoading(true);
    setError("");
    try {
      await api.resetPassword(token, password);
      navigate("/login");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Link inválido ou expirado.",
      );
    } finally {
      setLoading(false);
    }
  };
  return (
    <AuthLayout>
      <div className="auth-form">
        <p className="eyebrow">Acesso seguro</p>
        <h2>Defina uma nova senha</h2>
        <p className="muted">Use uma senha com pelo menos 8 caracteres.</p>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        <form onSubmit={submit}>
          <Field label="Nova senha">
            <Input
              required
              minLength={8}
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </Field>
          <Field label="Confirme a senha">
            <Input
              required
              minLength={8}
              type="password"
              value={confirmation}
              onChange={(e) => setConfirmation(e.target.value)}
            />
          </Field>
          <Button disabled={loading || !token}>
            {loading ? "Salvando..." : "Redefinir senha"}
            <ShieldCheck size={18} />
          </Button>
        </form>
      </div>
    </AuthLayout>
  );
}

function AcceptInvitePage() {
  const token = new URLSearchParams(window.location.search).get("token") ?? "";
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      await api.acceptInvite(token, name, password);
      navigate("/login");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Convite inválido ou expirado.",
      );
    } finally {
      setLoading(false);
    }
  };
  return (
    <AuthLayout>
      <div className="auth-form">
        <p className="eyebrow">Convite de equipe</p>
        <h2>Entre para a organização</h2>
        <p className="muted">
          Crie seu acesso para aceitar o convite recebido.
        </p>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        <form onSubmit={submit}>
          <Field label="Seu nome">
            <Input
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Ana Souza"
            />
          </Field>
          <Field label="Senha">
            <Input
              required
              minLength={8}
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </Field>
          <Button disabled={loading || !token}>
            {loading ? "Criando acesso..." : "Aceitar convite"}
            <ArrowRight size={18} />
          </Button>
        </form>
      </div>
    </AuthLayout>
  );
}

function AppSidebar({ clear }: { clear: () => void }) {
  const navItems = [
    { label: "Dashboard", to: "/app", icon: Grid2x2 },
    { label: "Campanhas", to: "/app/campaigns", icon: Megaphone },
    { label: "Templates", to: "/app/campaigns", icon: Files },
    { label: "Contatos", to: "/app/contacts", icon: List },
    { label: "Prévia", to: "/app/campaigns/preview", icon: Bell },
    { label: "Configurações", to: "/app/settings/email", icon: Settings },
  ];

  return (
    <aside className="app-sidebar">
      <div className="sidebar-header">
        <Logo />
        <button
          type="button"
          className="sidebar-menu-button"
          aria-label="Menu lateral"
        >
          <Menu size={18} />
        </button>
      </div>

      <nav className="sidebar-nav" aria-label="Navegação principal">
        {navItems.map(({ label, to, icon: Icon }) => (
          <NavLink
            key={label}
            className={({ isActive }) =>
              `sidebar-item ${isActive ? "active" : ""}`
            }
            to={to}
          >
            <Icon size={18} />
            <span>{label}</span>
          </NavLink>
        ))}
      </nav>

      <div className="sidebar-footer">
        <div className="profile-card">
          <div className="profile-avatar">
            <UserRound size={16} />
          </div>
          <div>
            <strong>Matheus Lima</strong>
            <span>Administrador</span>
          </div>
        </div>
        <button type="button" className="logout-button" onClick={clear}>
          <LogOut size={16} />
          Sair
        </button>
      </div>
    </aside>
  );
}

function AppHome({ session, clear }: { session: Session; clear: () => void }) {
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [stats, setStats] = useState<{
    sent: number;
    total: number;
    failed: number;
  } | null>(null);
  const [error, setError] = useState("");
  const load = async () => {
    try {
      const [nextIdentity, nextStats] = await Promise.all([
        api.me(session.access_token),
        api.deliveryStats(session.access_token),
      ]);
      setIdentity(nextIdentity);
      setStats(nextStats);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Falha ao carregar o dashboard.",
      );
    }
  };
  useEffect(() => {
    void load();
  }, [session.access_token]);
  const deliveryRate =
    stats && stats.total > 0
      ? `${Math.round((stats.sent / stats.total) * 100)}%`
      : "--";
  return (
    <div className="app-layout-shell">
      <AppSidebar clear={clear} />
      <main className="app-main-panel">
        <header className="topbar">
          <div className="topbar-title">
            <p className="eyebrow">Visão geral</p>
            <h1>Olá, Matheus!</h1>
          </div>
          <div className="topbar-actions">
            <button
              type="button"
              className="notification-button"
              aria-label="Notificações"
            >
              <Bell size={18} />
            </button>
            <button type="button" className="primary-action-button">
              + Nova Campanha
            </button>
          </div>
        </header>

        <div className="dashboard">
          <div className="dashboard-heading">
            <div>
              <p className="muted light-muted">
                Aqui está um resumo geral dos seus disparos de e-mails.
              </p>
            </div>
            <Button onClick={load}>
              Atualizar métricas <ArrowRight size={17} />
            </Button>
          </div>
          <section className="metric-grid">
            <article className="metric-card accent">
              <span className="metric-label">Próximo passo</span>
              <strong>Crie sua primeira campanha</strong>
              <p>Conecte seus contatos e prepare um envio memorável.</p>
              <Button
                className="button-light"
                onClick={() => window.location.assign("/app/campaigns")}
              >
                Começar agora <ArrowRight size={16} />
              </Button>
            </article>
            <article className="metric-card">
              <span className="metric-label">Mensagens enviadas</span>
              <strong>{stats?.sent ?? 0}</strong>
              <p>{stats?.failed ?? 0} falhas registradas.</p>
            </article>
            <article className="metric-card">
              <span className="metric-label">Taxa de entrega</span>
              <strong>{deliveryRate}</strong>
              <p>Baseado nos destinatários das campanhas.</p>
            </article>
          </section>
          {identity && (
            <div className="identity-card">
              <span className="metric-label">Sessão autenticada</span>
              <p>
                <b>Usuário:</b> {identity.user_id}
              </p>
              <p>
                <b>Organização:</b> {identity.organization_id}
              </p>
            </div>
          )}
          {error && <div className="error-box">{error}</div>}
        </div>
      </main>
    </div>
  );
}

function EmailSettingsPage({ session }: { session: Session }) {
  const navigate = useNavigate();
  const [settings, setSettings] = useState({
    provider: "smtp" as const,
    host: "",
    port: 587,
    username: "",
    password: "",
    from_email: "",
    from_name: "",
  });
  const [mfaCode, setMfaCode] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const save = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setMessage("");
    setLoading(true);
    try {
      await api.saveEmailSettings(session.access_token, mfaCode, settings);
      setMessage("Configuração protegida salva com sucesso.");
      setSettings({ ...settings, password: "" });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Não foi possível salvar.");
    } finally {
      setLoading(false);
    }
  };
  return (
    <div className="app-shell">
      <header className="topbar">
        <Logo />
        <Button className="button-ghost" onClick={() => navigate("/app")}>
          Voltar
        </Button>
      </header>
      <main className="settings-page">
        <p className="eyebrow">Canal de envio</p>
        <h1>Configurar e-mail</h1>
        <p className="muted">
          As credenciais ficam protegidas no backend e nunca retornam para esta
          tela.
        </p>
        <div className="security-banner">
          <ShieldCheck size={22} />
          <div>
            <b>Camada de segurança obrigatória</b>
            <span>
              Para salvar alterações, confirme sua identidade com MFA. A senha
              será criptografada antes de ser persistida.
            </span>
          </div>
        </div>
        {message && (
          <div className="success-box">
            <Check size={17} />
            {message}
          </div>
        )}
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        <form className="settings-form" onSubmit={save}>
          <div className="settings-fields">
            <Field label="Servidor SMTP">
              <Input
                required
                value={settings.host}
                onChange={(e) =>
                  setSettings({ ...settings, host: e.target.value })
                }
                placeholder="smtp.seuprovedor.com"
              />
            </Field>
            <Field label="Porta">
              <Input
                required
                type="number"
                min={1}
                max={65535}
                value={settings.port}
                onChange={(e) =>
                  setSettings({ ...settings, port: Number(e.target.value) })
                }
              />
            </Field>
            <Field label="Usuário SMTP">
              <Input
                required
                value={settings.username}
                onChange={(e) =>
                  setSettings({ ...settings, username: e.target.value })
                }
                placeholder="mailer@empresa.com"
              />
            </Field>
            <Field label="Senha SMTP">
              <Input
                required
                type="password"
                value={settings.password}
                onChange={(e) =>
                  setSettings({ ...settings, password: e.target.value })
                }
                placeholder="Nunca exibida depois de salva"
              />
            </Field>
            <Field label="E-mail remetente">
              <Input
                required
                type="email"
                value={settings.from_email}
                onChange={(e) =>
                  setSettings({ ...settings, from_email: e.target.value })
                }
                placeholder="contato@empresa.com"
              />
            </Field>
            <Field label="Nome remetente">
              <Input
                required
                value={settings.from_name}
                onChange={(e) =>
                  setSettings({ ...settings, from_name: e.target.value })
                }
                placeholder="Equipe NotifiQ"
              />
            </Field>
          </div>
          <div className="mfa-confirm">
            <Field label="Código MFA para autorizar">
              <Input
                required
                inputMode="numeric"
                pattern="[0-9]{6}"
                value={mfaCode}
                onChange={(e) => setMfaCode(e.target.value)}
                placeholder="000000"
              />
            </Field>
            <p>
              Essa confirmação é exigida em toda alteração de credenciais de
              envio.
            </p>
          </div>
          <Button disabled={loading}>
            {loading ? "Salvando..." : "Salvar configuração"}
            <ShieldCheck size={17} />
          </Button>
        </form>
      </main>
    </div>
  );
}

function ContactsPage({ session }: { session: Session }) {
  const navigate = useNavigate();
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const load = async () => {
    try {
      setContacts(await api.listContacts(session.access_token));
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível carregar os contatos.",
      );
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  const save = async (event: FormEvent) => {
    event.preventDefault();
    setError("");
    setSaving(true);
    try {
      const contact = await api.createContact(
        session.access_token,
        name,
        email,
      );
      setContacts([contact, ...contacts]);
      setName("");
      setEmail("");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível criar o contato.",
      );
    } finally {
      setSaving(false);
    }
  };
  return (
    <div className="app-shell">
      <header className="topbar">
        <Logo />
        <Button className="button-ghost" onClick={() => navigate("/app")}>
          Voltar
        </Button>
      </header>
      <main className="settings-page">
        <p className="eyebrow">Público</p>
        <h1>Seus contatos</h1>
        <p className="muted">
          Organize as pessoas que vão receber suas próximas mensagens.
        </p>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        <form className="settings-form contact-form" onSubmit={save}>
          <div className="settings-fields">
            <Field label="Nome">
              <Input
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Ana Souza"
              />
            </Field>
            <Field label="E-mail">
              <Input
                required
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="ana@empresa.com"
              />
            </Field>
          </div>
          <Button disabled={saving}>
            {saving ? "Adicionando..." : "Adicionar contato"}
            <ArrowRight size={17} />
          </Button>
        </form>
        <section className="contact-list">
          <div className="list-heading">
            <span className="metric-label">Lista atual</span>
            <span className="contact-count">{contacts.length} contatos</span>
          </div>
          {loading ? (
            <p className="muted">Carregando contatos...</p>
          ) : contacts.length === 0 ? (
            <p className="empty-state">
              Sua lista está vazia. Adicione o primeiro contato acima.
            </p>
          ) : (
            contacts.map((contact) => (
              <div className="contact-row" key={contact.id}>
                <span className="avatar">
                  {contact.name.charAt(0).toUpperCase()}
                </span>
                <div>
                  <b>{contact.name}</b>
                  <span>{contact.email}</span>
                </div>
              </div>
            ))
          )}
        </section>
      </main>
    </div>
  );
}

function ContactsImportPage({ session }: { session: Session }) {
  const navigate = useNavigate();
  const [file, setFile] = useState<File | null>(null);
  const [result, setResult] = useState<{
    imported: number;
    skipped: number;
    errors?: string[];
  } | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!file) {
      setError("Selecione um arquivo CSV.");
      return;
    }
    setLoading(true);
    setError("");
    setResult(null);
    try {
      setResult(await api.importContacts(session.access_token, file));
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível importar os contatos.",
      );
    } finally {
      setLoading(false);
    }
  };
  return (
    <div className="app-shell">
      <header className="topbar">
        <Logo />
        <Button
          className="button-ghost"
          onClick={() => navigate("/app/contacts")}
        >
          Voltar
        </Button>
      </header>
      <main className="settings-page">
        <p className="eyebrow">Público</p>
        <h1>Importar contatos</h1>
        <p className="muted">
          Use um CSV com as colunas <b>name,email</b>. Arquivos de até 5 MB são
          aceitos.
        </p>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        {result && (
          <div className="success-box">
            <Check size={17} />
            {result.imported} importados, {result.skipped} ignorados.
          </div>
        )}
        <form className="settings-form" onSubmit={submit}>
          <Field label="Arquivo CSV">
            <Input
              required
              type="file"
              accept=".csv,text/csv"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
          </Field>
          <Button disabled={loading}>
            {loading ? "Importando..." : "Importar contatos"}
            <ArrowRight size={17} />
          </Button>
        </form>
        {result?.errors && result.errors.length > 0 && (
          <section className="contact-list">
            <span className="metric-label">Linhas ignoradas</span>
            {result.errors.map((item) => (
              <p className="form-hint" key={item}>
                {item}
              </p>
            ))}
          </section>
        )}
      </main>
    </div>
  );
}

function CampaignManagePage({ session }: { session: Session }) {
  const navigate = useNavigate();
  const { campaignID } = useParams();
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [name, setName] = useState("");
  const [templateID, setTemplateID] = useState("");
  const [scheduledAt, setScheduledAt] = useState("");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);
  const campaign = campaigns.find((item) => item.id === campaignID);
  useEffect(() => {
    void Promise.all([
      api.listCampaigns(session.access_token),
      api.listTemplates(session.access_token),
    ])
      .then(([nextCampaigns, nextTemplates]) => {
        setCampaigns(nextCampaigns);
        setTemplates(nextTemplates);
        const current = nextCampaigns.find((item) => item.id === campaignID);
        if (current) {
          setName(current.name);
          setTemplateID(current.template_id);
          setScheduledAt(
            current.scheduled_at ? current.scheduled_at.slice(0, 16) : "",
          );
        }
      })
      .catch((err) =>
        setError(
          err instanceof Error
            ? err.message
            : "Não foi possível carregar a campanha.",
        ),
      );
  }, [session.access_token, campaignID]);
  const update = async (event: FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError("");
    setMessage("");
    try {
      await api.updateCampaign(
        session.access_token,
        campaignID ?? "",
        name,
        templateID,
        scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
      );
      setMessage("Campanha atualizada com sucesso.");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível atualizar a campanha.",
      );
    } finally {
      setLoading(false);
    }
  };
  const cancel = async () => {
    if (
      !window.confirm("Cancelar esta campanha e seus destinatários pendentes?")
    )
      return;
    setLoading(true);
    setError("");
    try {
      await api.cancelCampaign(session.access_token, campaignID ?? "");
      navigate("/app/campaigns");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível cancelar a campanha.",
      );
    } finally {
      setLoading(false);
    }
  };
  const duplicate = async () => {
    setLoading(true);
    setError("");
    try {
      const copy = await api.duplicateCampaign(
        session.access_token,
        campaignID ?? "",
      );
      navigate(`/app/campaigns/${copy.id}/manage`);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível duplicar a campanha.",
      );
    } finally {
      setLoading(false);
    }
  };
  return (
    <div className="app-shell">
      <header className="topbar">
        <Logo />
        <Button
          className="button-ghost"
          onClick={() => navigate("/app/campaigns")}
        >
          Voltar
        </Button>
      </header>
      <main className="settings-page">
        <p className="eyebrow">Orquestração</p>
        <h1>Gerenciar campanha</h1>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        {message && (
          <div className="success-box">
            <Check size={17} />
            {message}
          </div>
        )}
        {!campaign ? (
          <p className="empty-state">Campanha não encontrada.</p>
        ) : (
          <>
            <p className="muted">
              Status atual: <b>{campaign.status}</b>. Campanhas em envio não
              podem ser editadas.
            </p>
            <form className="settings-form" onSubmit={update}>
              <Field label="Nome">
                <Input
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </Field>
              <Field label="Template">
                <select
                  className="input"
                  required
                  value={templateID}
                  onChange={(e) => setTemplateID(e.target.value)}
                >
                  {templates.map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="Agendamento">
                <Input
                  type="datetime-local"
                  value={scheduledAt}
                  onChange={(e) => setScheduledAt(e.target.value)}
                />
              </Field>
              <Button
                disabled={
                  loading || !["draft", "scheduled"].includes(campaign.status)
                }
              >
                Salvar alterações <Check size={17} />
              </Button>
            </form>
            <div className="settings-actions">
              <Button
                className="button-light"
                disabled={loading}
                onClick={() => void duplicate()}
              >
                Duplicar campanha
              </Button>
              <Button
                className="button-danger"
                disabled={loading || campaign.status === "cancelled"}
                onClick={() => void cancel()}
              >
                Cancelar campanha
              </Button>
            </div>
          </>
        )}
      </main>
    </div>
  );
}

function CampaignPreviewPage({ session }: { session: Session }) {
  const navigate = useNavigate();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [templateID, setTemplateID] = useState("");
  const [error, setError] = useState("");
  useEffect(() => {
    void api
      .listTemplates(session.access_token)
      .then((items) => {
        setTemplates(items);
        if (items[0]) setTemplateID(items[0].id);
      })
      .catch((err) =>
        setError(
          err instanceof Error
            ? err.message
            : "Não foi possível carregar os templates.",
        ),
      );
  }, [session.access_token]);
  const template = templates.find((item) => item.id === templateID);
  const previewHTML =
    template?.body_html
      .replaceAll("{{nome}}", "Ana Souza")
      .replaceAll("{{email}}", "cliente@example.com") ??
    "<p>Selecione um template para visualizar.</p>";
  return (
    <div className="app-shell">
      <header className="topbar">
        <Logo />
        <Button
          className="button-ghost"
          onClick={() => navigate("/app/campaigns")}
        >
          Voltar
        </Button>
      </header>
      <main className="settings-page">
        <p className="eyebrow">Revisão</p>
        <h1>Pré-visualizar campanha</h1>
        <p className="muted">
          Veja uma simulação segura de como a mensagem será exibida ao
          destinatário.
        </p>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        <div className="preview-toolbar">
          <Field label="Template">
            <select
              className="input"
              value={templateID}
              onChange={(e) => setTemplateID(e.target.value)}
            >
              {templates.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
          </Field>
          {template && (
            <div className="preview-subject">
              <span className="metric-label">Assunto</span>
              <b>{template.subject.replaceAll("{{nome}}", "Ana Souza")}</b>
            </div>
          )}
        </div>
        <section className="email-preview">
          <div className="preview-browser">
            <span />
            <span />
            <span />
          </div>
          <iframe
            title="Pré-visualização do e-mail"
            sandbox=""
            srcDoc={previewHTML}
          />
        </section>
      </main>
    </div>
  );
}

function buildGuidedTemplateHtml({
  headline,
  bodyText,
  ctaLabel,
  ctaHref,
  imageUrl,
}: {
  headline: string;
  bodyText: string;
  ctaLabel: string;
  ctaHref: string;
  imageUrl: string;
}) {
  const safeHeadline = headline.trim() || "Olá, {{nome}}!";
  const safeBody = (
    bodyText.trim() || "Seu conteúdo vai aparecer aqui."
  ).replace(/\n/g, "<br />");
  const safeCtaLabel = ctaLabel.trim() || "Acessar agora";
  const safeCtaHref = ctaHref.trim() || "#";
  const banner = imageUrl.trim()
    ? `<p style="margin:0 0 18px;"><img src="${imageUrl.trim()}" alt="Imagem do template" style="max-width:100%;border-radius:12px;display:block;" /></p>`
    : "";

  return `
    <div style="font-family: Arial, sans-serif; line-height: 1.6; color: #17231f; background: #ffffff; padding: 28px; border-radius: 16px; border: 1px solid #e7e2d9;">
      ${banner}
      <h1 style="margin:0 0 12px; font-size: 28px; line-height: 1.2; letter-spacing: -0.04em; color: #17352d;">${safeHeadline}</h1>
      <p style="margin:0 0 18px; font-size: 16px; color: #405149;">${safeBody}</p>
      <p style="margin: 0;">
        <a href="${safeCtaHref}" style="display:inline-block; padding: 12px 20px; background: #17352d; color: #ffffff; text-decoration: none; border-radius: 999px; font-weight: 700; font-size: 14px;">${safeCtaLabel}</a>
      </p>
      <p style="margin: 20px 0 0; font-size: 12px; color: #718079;">Atenciosamente,<br />Equipe Notifiq</p>
    </div>
  `;
}

function CampaignsPage({ session }: { session: Session }) {
  const navigate = useNavigate();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [selectedContacts, setSelectedContacts] = useState<string[]>([]);
  const [name, setName] = useState("");
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [campaignName, setCampaignName] = useState("");
  const [templateID, setTemplateID] = useState("");
  const [scheduledAt, setScheduledAt] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [imageUrl, setImageUrl] = useState("");
  const [templateMode, setTemplateMode] = useState<"guided" | "html">("guided");
  const [headline, setHeadline] = useState("Olá, {{nome}}!");
  const [bodyText, setBodyText] = useState(
    "Estamos muito felizes em ter você conosco e queremos te manter informado.",
  );
  const [ctaLabel, setCtaLabel] = useState("Acessar agora");
  const [ctaHref, setCtaHref] = useState("#");
  const templatePreviewHtml = buildGuidedTemplateHtml({
    headline,
    bodyText,
    ctaLabel,
    ctaHref,
    imageUrl,
  });
  useEffect(() => {
    void Promise.all([
      api.listTemplates(session.access_token),
      api.listCampaigns(session.access_token),
      api.listContacts(session.access_token),
    ])
      .then(([nextTemplates, nextCampaigns, nextContacts]) => {
        setTemplates(nextTemplates);
        setCampaigns(nextCampaigns);
        setContacts(nextContacts);
        if (nextTemplates[0]) setTemplateID(nextTemplates[0].id);
      })
      .catch((err) =>
        setError(
          err instanceof Error
            ? err.message
            : "Não foi possível carregar campanhas.",
        ),
      );
  }, [session.access_token]);
  const createTemplate = async (event: FormEvent) => {
    event.preventDefault();
    setSaving(true);
    setError("");
    try {
      const composedBody =
        templateMode === "guided"
          ? buildGuidedTemplateHtml({
              headline,
              bodyText,
              ctaLabel,
              ctaHref,
              imageUrl,
            })
          : imageUrl.trim()
            ? `<p><img src="${imageUrl.trim()}" alt="Imagem do template" style="max-width:100%;height:auto;display:block;margin:0 0 16px;" /></p>${body}`
            : body;
      const item = await api.createTemplate(
        session.access_token,
        name,
        subject,
        composedBody,
      );
      setTemplates([item, ...templates]);
      setTemplateID(item.id);
      setName("");
      setSubject("");
      setBody("");
      setImageUrl("");
      setHeadline("Olá, {{nome}}!");
      setBodyText(
        "Estamos muito felizes em ter você conosco e queremos te manter informado.",
      );
      setCtaLabel("Acessar agora");
      setCtaHref("#");
      setTemplateMode("guided");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível criar o template.",
      );
    } finally {
      setSaving(false);
    }
  };
  const deleteTemplate = async (templateId: string) => {
    if (
      !window.confirm(
        "Excluir este template? Esta ação remove o conteúdo da organização.",
      )
    )
      return;
    setError("");
    try {
      await api.deleteTemplate(session.access_token, templateId);
      setTemplates((current) =>
        current.filter((item) => item.id !== templateId),
      );
      setTemplateID((current) => (current === templateId ? "" : current));
      if (templateID === templateId) setCampaignName("");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível excluir o template.",
      );
    }
  };

  const deleteCampaign = async (campaignId: string) => {
    if (
      !window.confirm(
        "Excluir esta campanha? Todos os registros de destinatários serão removidos.",
      )
    )
      return;
    setError("");
    try {
      await api.deleteCampaign(session.access_token, campaignId);
      setCampaigns((current) =>
        current.filter((item) => item.id !== campaignId),
      );
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível excluir a campanha.",
      );
    }
  };

  const createCampaign = async (event: FormEvent) => {
    event.preventDefault();
    setSaving(true);
    setError("");
    try {
      const item = await api.createCampaign(
        session.access_token,
        campaignName,
        templateID,
        scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
      );
      if (selectedContacts.length > 0)
        await api.addCampaignRecipients(
          session.access_token,
          item.id,
          selectedContacts,
        );
      setCampaigns([item, ...campaigns]);
      setCampaignName("");
      setScheduledAt("");
      setSelectedContacts([]);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível criar a campanha.",
      );
    } finally {
      setSaving(false);
    }
  };
  const toggleContact = (id: string) =>
    setSelectedContacts((current) =>
      current.includes(id)
        ? current.filter((item) => item !== id)
        : [...current, id],
    );
  return (
    <div className="app-shell">
      <header className="topbar">
        <Logo />
        <Button className="button-ghost" onClick={() => navigate("/app")}>
          Voltar
        </Button>
      </header>
      <main className="settings-page">
        <p className="eyebrow">Orquestração</p>
        <h1>Campanhas</h1>
        <p className="muted">
          Prepare o conteúdo agora. O disparo acontece em uma próxima etapa
          controlada.
        </p>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        <div className="campaign-columns">
          <form className="settings-form" onSubmit={createTemplate}>
            <span className="metric-label">Novo template</span>
            <div className="template-editor-header">
              <Field label="Nome">
                <Input
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Boas-vindas"
                />
              </Field>
              <Field label="Assunto">
                <Input
                  required
                  value={subject}
                  onChange={(e) => setSubject(e.target.value)}
                  placeholder="Olá, {{nome}}"
                />
              </Field>
            </div>

            <div className="template-mode-switch">
              <button
                type="button"
                className={
                  templateMode === "guided" ? "mode-pill active" : "mode-pill"
                }
                onClick={() => setTemplateMode("guided")}
              >
                Simples
              </button>
              <button
                type="button"
                className={
                  templateMode === "html" ? "mode-pill active" : "mode-pill"
                }
                onClick={() => setTemplateMode("html")}
              >
                HTML avançado
              </button>
            </div>

            {templateMode === "guided" ? (
              <>
                <Field label="Imagem (opcional)">
                  <Input
                    type="url"
                    value={imageUrl}
                    onChange={(e) => setImageUrl(e.target.value)}
                    placeholder="https://exemplo.com/banner.png"
                  />
                </Field>
                <div className="template-quick-fields">
                  <Field label="Título">
                    <Input
                      value={headline}
                      onChange={(e) => setHeadline(e.target.value)}
                      placeholder="Olá, {{nome}}!"
                    />
                  </Field>
                  <Field label="Botão">
                    <Input
                      value={ctaLabel}
                      onChange={(e) => setCtaLabel(e.target.value)}
                      placeholder="Acessar agora"
                    />
                  </Field>
                </div>
                <Field label="Texto principal">
                  <textarea
                    className="textarea"
                    value={bodyText}
                    onChange={(e) => setBodyText(e.target.value)}
                    placeholder="Escreva a mensagem principal do e-mail"
                  />
                </Field>
                <Field label="Link do botão">
                  <Input
                    type="url"
                    value={ctaHref}
                    onChange={(e) => setCtaHref(e.target.value)}
                    placeholder="https://exemplo.com/acao"
                  />
                </Field>
                <div className="template-preview-card">
                  <span className="metric-label">Pré-visualização</span>
                  <div
                    className="template-preview-content"
                    dangerouslySetInnerHTML={{ __html: templatePreviewHtml }}
                  />
                </div>
              </>
            ) : (
              <>
                <Field label="Imagem (opcional)">
                  <Input
                    type="url"
                    value={imageUrl}
                    onChange={(e) => setImageUrl(e.target.value)}
                    placeholder="https://exemplo.com/banner.png"
                  />
                </Field>
                <Field label="Conteúdo HTML">
                  <textarea
                    required
                    className="textarea"
                    value={body}
                    onChange={(e) => setBody(e.target.value)}
                    placeholder="<h1>Olá!</h1>"
                  />
                </Field>
              </>
            )}
            <Button disabled={saving}>
              Salvar template <Check size={17} />
            </Button>
          </form>
          <form className="settings-form" onSubmit={createCampaign}>
            <span className="metric-label">Nova campanha</span>
            <Field label="Nome da campanha">
              <Input
                required
                value={campaignName}
                onChange={(e) => setCampaignName(e.target.value)}
                placeholder="Lançamento de agosto"
              />
            </Field>
            <Field label="Template">
              <select
                className="input"
                required
                value={templateID}
                onChange={(e) => setTemplateID(e.target.value)}
              >
                <option value="">Selecione um template</option>
                {templates.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Agendar (opcional)">
              <Input
                type="datetime-local"
                value={scheduledAt}
                onChange={(e) => setScheduledAt(e.target.value)}
              />
            </Field>
            <div className="recipient-picker">
              <div className="list-heading">
                <span className="metric-label">Destinatários</span>
                <span className="contact-count">
                  {selectedContacts.length} selecionados
                </span>
              </div>
              {contacts.length === 0 ? (
                <p className="form-hint">Cadastre contatos antes de enviar.</p>
              ) : (
                contacts.map((contact) => (
                  <label className="recipient-option" key={contact.id}>
                    <input
                      type="checkbox"
                      checked={selectedContacts.includes(contact.id)}
                      onChange={() => toggleContact(contact.id)}
                    />
                    <span>{contact.name}</span>
                    <small>{contact.email}</small>
                  </label>
                ))
              )}
            </div>
            <Button disabled={saving || templates.length === 0}>
              Criar campanha <ArrowRight size={17} />
            </Button>
            {templates.length === 0 && (
              <p className="form-hint">
                Crie um template antes de criar uma campanha.
              </p>
            )}
          </form>
        </div>
        <section className="contact-list">
          <div className="list-heading">
            <span className="metric-label">Templates salvos</span>
            <span className="contact-count">{templates.length} templates</span>
          </div>
          {templates.length === 0 ? (
            <p className="empty-state">Nenhum template criado ainda.</p>
          ) : (
            templates.map((item) => (
              <div className="contact-row" key={item.id}>
                <div>
                  <b>{item.name}</b>
                  <span>{item.subject}</span>
                </div>
                <Button
                  className="button-ghost"
                  onClick={() => void deleteTemplate(item.id)}
                >
                  Excluir
                </Button>
              </div>
            ))
          )}
        </section>
        <section className="contact-list">
          <div className="list-heading">
            <span className="metric-label">Campanhas recentes</span>
            <span className="contact-count">{campaigns.length} campanhas</span>
          </div>
          {campaigns.length === 0 ? (
            <p className="empty-state">Nenhuma campanha criada ainda.</p>
          ) : (
            campaigns.map((item) => (
              <div className="contact-row" key={item.id}>
                <Link
                  className="contact-row"
                  to={`/app/campaigns/${item.id}/history`}
                >
                  <span className={`status-badge ${item.status}`}>
                    {item.status}
                  </span>
                  <div>
                    <b>{item.name}</b>
                    <span>
                      {item.status === "scheduled" && item.scheduled_at
                        ? `Agendada para ${new Date(item.scheduled_at).toLocaleString("pt-BR")}`
                        : "Ver histórico de entrega"}
                    </span>
                  </div>
                  <ArrowRight size={16} />
                </Link>
                <Button
                  className="button-ghost"
                  onClick={() => void deleteCampaign(item.id)}
                >
                  Excluir
                </Button>
              </div>
            ))
          )}
        </section>
      </main>
    </div>
  );
}

function CampaignHistoryPage({ session }: { session: Session }) {
  const navigate = useNavigate();
  const { campaignID } = useParams();
  const [history, setHistory] = useState<import("./lib/api").DeliveryHistory[]>(
    [],
  );
  const [error, setError] = useState("");
  useEffect(() => {
    if (!campaignID) return;
    void api
      .deliveryHistory(session.access_token, campaignID)
      .then(setHistory)
      .catch((err) =>
        setError(
          err instanceof Error
            ? err.message
            : "Não foi possível carregar o histórico.",
        ),
      );
  }, [session.access_token, campaignID]);
  return (
    <div className="app-shell">
      <header className="topbar">
        <Logo />
        <Button
          className="button-ghost"
          onClick={() => navigate("/app/campaigns")}
        >
          Voltar
        </Button>
      </header>
      <main className="settings-page">
        <p className="eyebrow">Observabilidade</p>
        <h1>Histórico de entrega</h1>
        <p className="muted">
          Acompanhe cada destinatário e os eventos recebidos do provedor.
        </p>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        <section className="contact-list">
          {history.length === 0 ? (
            <p className="empty-state">
              Nenhum destinatário encontrado para esta campanha.
            </p>
          ) : (
            history.map((item) => (
              <article
                className="contact-row history-row"
                key={item.contact_id}
              >
                <span className={`status-badge ${item.status}`}>
                  {item.status}
                </span>
                <div>
                  <b>{item.contact_name}</b>
                  <span>
                    {item.email} · {item.attempts} tentativa(s)
                  </span>
                  {item.events.length > 0 && (
                    <small>
                      {item.events.map((event) => event.type).join(" · ")}
                    </small>
                  )}
                </div>
              </article>
            ))
          )}
        </section>
      </main>
    </div>
  );
}

function InviteMemberForm({ session }: { session: Session }) {
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<Member["role"]>("viewer");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError("");
    setMessage("");
    try {
      await api.inviteMember(session.access_token, email, role);
      setMessage("Convite enviado com segurança.");
      setEmail("");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível enviar o convite.",
      );
    } finally {
      setLoading(false);
    }
  };
  return (
    <form className="settings-form" onSubmit={submit}>
      <span className="metric-label">Convidar pessoa</span>
      {message && (
        <div className="success-box">
          <Check size={17} />
          {message}
        </div>
      )}
      {error && (
        <div className="error-box">
          <CircleAlert size={17} />
          {error}
        </div>
      )}
      <div className="settings-fields">
        <Field label="E-mail">
          <Input
            required
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="novo@empresa.com"
          />
        </Field>
        <Field label="Permissão">
          <select
            className="input"
            value={role}
            onChange={(e) => setRole(e.target.value as Member["role"])}
          >
            <option value="viewer">Viewer</option>
            <option value="editor">Editor</option>
            <option value="admin">Admin</option>
          </select>
        </Field>
      </div>
      <Button disabled={loading}>
        {loading ? "Enviando..." : "Enviar convite"}
        <ArrowRight size={17} />
      </Button>
    </form>
  );
}

function MembersPage({ session }: { session: Session }) {
  const navigate = useNavigate();
  const [members, setMembers] = useState<Member[]>([]);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState("");
  const load = async () => {
    try {
      setMembers(await api.listMembers(session.access_token));
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível carregar a equipe.",
      );
    }
  };
  useEffect(() => {
    void load();
  }, [session.access_token]);
  const changeRole = async (member: Member, role: Member["role"]) => {
    setSaving(member.user_id);
    setError("");
    try {
      await api.updateMemberRole(session.access_token, member.user_id, role);
      setMembers(
        members.map((item) =>
          item.user_id === member.user_id ? { ...item, role } : item,
        ),
      );
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível alterar o papel.",
      );
    } finally {
      setSaving("");
    }
  };
  const remove = async (member: Member) => {
    if (!window.confirm(`Remover ${member.name} da organização?`)) return;
    setSaving(member.user_id);
    setError("");
    try {
      await api.removeMember(session.access_token, member.user_id);
      setMembers(members.filter((item) => item.user_id !== member.user_id));
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Não foi possível remover o membro.",
      );
    } finally {
      setSaving("");
    }
  };
  return (
    <div className="app-shell">
      <header className="topbar">
        <Logo />
        <Button className="button-ghost" onClick={() => navigate("/app")}>
          Voltar
        </Button>
      </header>
      <main className="settings-page">
        <p className="eyebrow">Organização</p>
        <h1>Equipe e permissões</h1>
        <p className="muted">
          Controle quem pode criar campanhas, configurar envios e administrar o
          espaço.
        </p>
        {error && (
          <div className="error-box">
            <CircleAlert size={17} />
            {error}
          </div>
        )}
        <section className="contact-list">
          <div className="list-heading">
            <span className="metric-label">Membros atuais</span>
            <span className="contact-count">{members.length} pessoas</span>
          </div>
          {members.length === 0 ? (
            <p className="empty-state">Nenhum membro encontrado.</p>
          ) : (
            members.map((member) => (
              <div className="contact-row member-row" key={member.user_id}>
                <div className="avatar">
                  {member.name.charAt(0).toUpperCase()}
                </div>
                <div>
                  <b>{member.name}</b>
                  <span>{member.email}</span>
                </div>
                <select
                  className="input role-select"
                  value={member.role}
                  disabled={saving === member.user_id}
                  onChange={(event) =>
                    void changeRole(
                      member,
                      event.target.value as Member["role"],
                    )
                  }
                >
                  <option value="owner">Owner</option>
                  <option value="admin">Admin</option>
                  <option value="editor">Editor</option>
                  <option value="viewer">Viewer</option>
                </select>
                <Button
                  className="button-ghost"
                  disabled={
                    saving === member.user_id || member.role === "owner"
                  }
                  onClick={() => void remove(member)}
                >
                  Remover
                </Button>
              </div>
            ))
          )}
        </section>
      </main>
    </div>
  );
}

function MfaSetupRoute({ session }: { session: Session }) {
  const navigate = useNavigate();
  return <MfaSetupPage session={session} onComplete={() => navigate("/app")} />;
}

export function App() {
  const auth = useSession();
  return (
    <Routes>
      <Route
        path="/login"
        element={
          auth.session ? (
            <Navigate to="/app" />
          ) : (
            <AuthForm mode="login" onAuth={auth.save} />
          )
        }
      />
      <Route
        path="/register"
        element={
          auth.session ? (
            <Navigate to="/app" />
          ) : (
            <AuthForm mode="register" onAuth={auth.save} />
          )
        }
      />
      <Route
        path="/forgot-password"
        element={
          auth.session ? <Navigate to="/app" /> : <PasswordResetRequestPage />
        }
      />
      <Route
        path="/reset-password"
        element={auth.session ? <Navigate to="/app" /> : <ResetPasswordPage />}
      />
      <Route
        path="/invite"
        element={auth.session ? <Navigate to="/app" /> : <AcceptInvitePage />}
      />
      <Route
        path="/app"
        element={
          auth.session ? (
            <AppHome session={auth.session} clear={auth.clear} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="/app/mfa-setup"
        element={
          auth.session ? (
            <MfaSetupRoute session={auth.session} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="/app/settings/email"
        element={
          auth.session ? (
            <EmailSettingsPage session={auth.session} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="/app/contacts"
        element={
          auth.session ? (
            <ContactsPage session={auth.session} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="/app/contacts/import"
        element={
          auth.session ? (
            <ContactsImportPage session={auth.session} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="/app/campaigns"
        element={
          auth.session ? (
            <CampaignsPage session={auth.session} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="/app/campaigns/preview"
        element={
          auth.session ? (
            <CampaignPreviewPage session={auth.session} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="/app/campaigns/:campaignID/history"
        element={
          auth.session ? (
            <CampaignHistoryPage session={auth.session} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="/app/campaigns/:campaignID/manage"
        element={
          auth.session ? (
            <CampaignManagePage session={auth.session} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="/app/team"
        element={
          auth.session ? (
            <MembersPage session={auth.session} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="/app/team/invite"
        element={
          auth.session ? (
            <InviteMemberForm session={auth.session} />
          ) : (
            <Navigate to="/login" />
          )
        }
      />
      <Route
        path="*"
        element={<Navigate to={auth.session ? "/app" : "/login"} />}
      />
    </Routes>
  );
}
