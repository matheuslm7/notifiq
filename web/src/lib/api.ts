export type Session = {
  access_token: string;
  refresh_token: string;
  expires_at: string;
};

export type Identity = {
  user_id: string;
  organization_id: string;
};

export type EmailSettings = {
  provider: "smtp";
  host: string;
  port: number;
  username: string;
  password?: string;
  from_email: string;
  from_name: string;
  has_password?: boolean;
};
export type Contact = {
  id: string;
  name: string;
  email: string;
  created_at: string;
};
export type Template = {
  id: string;
  name: string;
  subject: string;
  body_html: string;
};
export type Campaign = {
  id: string;
  template_id: string;
  name: string;
  status: string;
  scheduled_at?: string;
};
export type DeliveryStats = {
  total: number;
  pending: number;
  sending: number;
  sent: number;
  failed: number;
  cancelled: number;
};
export type DeliveryEvent = {
  id: string;
  provider_event_id: string;
  provider_message_id?: string;
  type: string;
  occurred_at?: string;
};
export type DeliveryHistory = {
  campaign_id: string;
  contact_id: string;
  contact_name: string;
  email: string;
  status: string;
  attempts: number;
  sent_at?: string;
  events: DeliveryEvent[];
};
export type Member = {
  user_id: string;
  organization_id: string;
  name: string;
  email: string;
  role: "owner" | "admin" | "editor" | "viewer";
  created_at: string;
};
export type MFASetup = { secret: string; otpauth_url: string };

const API_PREFIX = import.meta.env.VITE_API_URL ?? "/api";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_PREFIX}${path}`, {
    headers: { "Content-Type": "application/json", ...init?.headers },
    ...init,
  });
  const body = (await response.json().catch(() => ({}))) as {
    error?: string;
  } & T;
  if (!response.ok) {
    throw new Error(body.error ?? "Não foi possível concluir a solicitação.");
  }
  return body;
}

export const api = {
  register: (name: string, email: string, password: string) =>
    request<Session>("/auth/register", {
      method: "POST",
      body: JSON.stringify({ name, email, password }),
    }),
  login: (email: string, password: string, mfa_code?: string) =>
    request<Session>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password, mfa_code }),
    }),
  setupMFA: (token: string) =>
    request<MFASetup>("/auth/mfa/setup", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
    }),
  confirmMFA: (token: string, code: string) =>
    request<void>("/auth/mfa/confirm", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ code }),
    }),
  me: (token: string) =>
    request<Identity>("/me", { headers: { Authorization: `Bearer ${token}` } }),
  getEmailSettings: (token: string) =>
    request<EmailSettings>("/settings/email", {
      headers: { Authorization: `Bearer ${token}` },
    }),
  saveEmailSettings: (
    token: string,
    mfaCode: string,
    settings: EmailSettings,
  ) =>
    request<void>("/settings/email/save", {
      method: "PUT",
      headers: { Authorization: `Bearer ${token}`, "X-MFA-Code": mfaCode },
      body: JSON.stringify(settings),
    }),
  listContacts: (token: string) =>
    request<Contact[]>("/contacts", {
      headers: { Authorization: `Bearer ${token}` },
    }),
  createContact: (token: string, name: string, email: string) =>
    request<Contact>("/contacts/create", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ name, email }),
    }),
  listTemplates: (token: string) =>
    request<Template[]>("/templates", {
      headers: { Authorization: `Bearer ${token}` },
    }),
  createTemplate: (
    token: string,
    name: string,
    subject: string,
    body_html: string,
  ) =>
    request<Template>("/templates", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ name, subject, body_html }),
    }),
  deleteTemplate: (token: string, template_id: string) =>
    request<void>("/templates", {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ template_id }),
    }),
  listCampaigns: (token: string) =>
    request<Campaign[]>("/campaigns", {
      headers: { Authorization: `Bearer ${token}` },
    }),
  createCampaign: (
    token: string,
    name: string,
    template_id: string,
    scheduled_at?: string,
  ) =>
    request<Campaign>("/campaigns", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ name, template_id, scheduled_at }),
    }),
  deleteCampaign: (token: string, campaign_id: string) =>
    request<void>("/campaigns", {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ campaign_id }),
    }),
  addCampaignRecipients: (
    token: string,
    campaign_id: string,
    contact_ids: string[],
  ) =>
    request<void>("/campaign-recipients", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ campaign_id, contact_ids }),
    }),
  deliveryStats: (token: string) =>
    request<DeliveryStats>("/metrics/delivery", {
      headers: { Authorization: `Bearer ${token}` },
    }),
  deliveryHistory: (token: string, campaignID: string) =>
    request<DeliveryHistory[]>(
      `/metrics/delivery/history?campaign_id=${encodeURIComponent(campaignID)}`,
      { headers: { Authorization: `Bearer ${token}` } },
    ),
  listMembers: (token: string) =>
    request<Member[]>("/members", {
      headers: { Authorization: `Bearer ${token}` },
    }),
  updateMemberRole: (token: string, member_id: string, role: Member["role"]) =>
    request<void>("/members", {
      method: "PUT",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ member_id, role }),
    }),
  removeMember: (token: string, member_id: string) =>
    request<void>("/members/remove", {
      method: "DELETE",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ member_id }),
    }),
  requestPasswordReset: (email: string) =>
    request<void>("/auth/password-reset/request", {
      method: "POST",
      body: JSON.stringify({ email }),
    }),
  resetPassword: (token: string, password: string) =>
    request<void>("/auth/password-reset/confirm", {
      method: "POST",
      body: JSON.stringify({ token, password }),
    }),
  changePassword: (
    token: string,
    current_password: string,
    new_password: string,
  ) =>
    request<void>("/auth/password/change", {
      method: "PUT",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ current_password, new_password }),
    }),
  inviteMember: (token: string, email: string, role: Member["role"]) =>
    request<void>("/members/invite", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ email, role }),
    }),
  acceptInvite: (token: string, name: string, password: string) =>
    request<Member>("/members/invite/accept", {
      method: "POST",
      body: JSON.stringify({ token, name, password }),
    }),
  importContacts: async (token: string, file: File) => {
    const form = new FormData();
    form.append("file", file);
    const response = await fetch(`${API_PREFIX}/contacts/import`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: form,
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok)
      throw new Error(body.error ?? "Não foi possível importar os contatos.");
    return body as { imported: number; skipped: number; errors?: string[] };
  },
  updateCampaign: (
    token: string,
    campaign_id: string,
    name: string,
    template_id: string,
    scheduled_at?: string,
  ) =>
    request<void>("/campaigns/update", {
      method: "PUT",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ campaign_id, name, template_id, scheduled_at }),
    }),
  cancelCampaign: (token: string, campaign_id: string) =>
    request<void>("/campaigns/cancel", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ campaign_id }),
    }),
  duplicateCampaign: (token: string, campaign_id: string) =>
    request<Campaign>("/campaigns/duplicate", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: JSON.stringify({ campaign_id }),
    }),
};
