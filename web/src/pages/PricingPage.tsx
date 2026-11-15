import {
  ArrowRight,
  ArrowUpRight,
  Check,
  MessageSquare,
  ShieldCheck,
  Terminal,
  Zap,
} from 'lucide-react'
import { Link } from 'react-router'
import HeaderBar from '../components/common/HeaderBar'
import FooterSection from '../components/landing/FooterSection'
import LoginModal from '../components/landing/LoginModal'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import { OFFICIAL_LINKS } from '../constants/branding'
import { ROUTES } from '../router/paths'
import { useState } from 'react'

function copy(lang: string, zh: string, en: string, id?: string) {
  if (lang === 'zh') return zh
  if (lang === 'id' && id) return id
  return en
}

type TierId = 'community' | 'cloud' | 'pro' | 'team'

interface Tier {
  id: TierId
  badge?: string
  name: string
  price: string
  period: string
  blurb: string
  highlights: string[]
  cta: string
  to?: string
  href?: string
  featured?: boolean
  note?: string
}

export function PricingPage() {
  const [showLoginModal, setShowLoginModal] = useState(false)
  const { user, token, logout } = useAuth()
  const { language, setLanguage } = useLanguage()
  const isLoggedIn = Boolean(user && token)

  const tiers: Tier[] = [
    {
      id: 'community',
      name: copy(language, 'Community', 'Community', 'Community'),
      price: copy(language, '免费', '$0', 'Gratis'),
      period: copy(
        language,
        '自托管开源',
        'Self-host forever',
        'Self-host selamanya'
      ),
      blurb: copy(
        language,
        '完整终端在你自己的机器上跑。适合能 Docker / 本机部署、要密钥不出机的人。',
        'Full terminal on your machine. For people who can Docker or self-host and want keys to stay local.',
        'Terminal penuh di mesin Anda. Untuk yang bisa Docker/self-host dan ingin kunci tetap lokal.'
      ),
      highlights: [
        copy(language, 'Markets / Trade / Autopilot / Portfolio', 'Markets / Trade / Autopilot / Portfolio', 'Markets / Trade / Autopilot / Portfolio'),
        copy(language, 'Claw402 方向板 + Preflight + 硬风控闸', 'Claw402 board + Preflight + hard gates', 'Papan Claw402 + Preflight + hard gate'),
        copy(language, '多交易所（HL / CEX / Lighter / Aster）', 'Multi-venue (HL / CEX / Lighter / Aster)', 'Multi-venue (HL / CEX / Lighter / Aster)'),
        copy(language, 'Messaging 多 IM 绑定 + 告警', 'Multi-IM messaging + alerts', 'Messaging multi-IM + peringatan'),
        copy(language, 'Web Chat 只读行情 Copilot', 'Web Chat read-only market Copilot', 'Web Chat Copilot pasar baca-saja'),
        copy(language, 'AI 用量按 Claw402 / BYOK 自付', 'AI usage via Claw402 / BYOK', 'Pemakaian AI via Claw402 / BYOK'),
      ],
      cta: copy(language, '打开 Markets 开始', 'Open Markets to start', 'Buka Markets untuk mulai'),
      to: ROUTES.markets,
    },
    {
      id: 'cloud',
      badge: copy(language, '即将开放', 'Coming soon', 'Segera'),
      name: copy(language, 'Cloud', 'Cloud', 'Cloud'),
      price: '$49',
      period: copy(language, '/ 月 · 示意', '/ mo · indicative', '/ bln · indikatif'),
      blurb: copy(
        language,
        '官方托管实例：少运维，保留非托管交易账户模型。适合想用产品、不想自己盯服务器的人。',
        'Hosted instance with less ops, same non-custodial trading-account model. For people who want the product without running servers.',
        'Instans hosted, lebih sedikit ops, model akun non-kustodi sama.'
      ),
      highlights: [
        copy(language, 'Community 全部能力', 'Everything in Community', 'Semua di Community'),
        copy(language, '托管部署、备份与升级', 'Hosted deploy, backup & upgrades', 'Deploy, backup & upgrade terkelola'),
        copy(language, '预装 Messaging 告警通道', 'Messaging alerts pre-wired', 'Peringatan Messaging siap pakai'),
        copy(language, '基础邮件 / 社区支持', 'Basic email / community support', 'Dukungan email / komunitas dasar'),
      ],
      cta: copy(language, '加入候补 / 社区', 'Join waitlist / community', 'Gabung waitlist / komunitas'),
      href: OFFICIAL_LINKS.telegram || ROUTES.register,
      featured: true,
      note: copy(
        language,
        '示意价；上线后以正式页为准。交易保证金仍在你的交易所 / 钱包。',
        'Indicative price; final pricing at launch. Trading margin stays on your exchange / wallet.',
        'Harga indikatif; final saat peluncuran. Margin tetap di bursa/dompet Anda.'
      ),
    },
    {
      id: 'pro',
      badge: copy(language, '即将开放', 'Coming soon', 'Segera'),
      name: copy(language, 'Pro', 'Pro', 'Pro'),
      price: '$149',
      period: copy(language, '/ 月 · 示意', '/ mo · indicative', '/ bln · indikatif'),
      blurb: copy(
        language,
        '给要跑多 Autopilot、更细风控模板与优先支持的进阶玩家。',
        'For power users running multiple Autopilots, richer risk templates, and priority support.',
        'Untuk power user dengan banyak Autopilot, template risiko lebih kaya, dan dukungan prioritas.'
      ),
      highlights: [
        copy(language, 'Cloud 全部能力', 'Everything in Cloud', 'Semua di Cloud'),
        copy(language, '多策略 / 多实例额度更高', 'Higher multi-strategy / multi-instance limits', 'Batas multi-strategi / multi-instans lebih tinggi'),
        copy(language, 'Leaderboard 与竞赛高级能力', 'Advanced Leaderboard / competition tools', 'Alat Leaderboard / kompetisi lanjutan'),
        copy(language, '优先支持与启动协助', 'Priority support & launch help', 'Dukungan prioritas & bantuan peluncuran'),
      ],
      cta: copy(language, '聊聊 Pro 需求', 'Talk about Pro', 'Bicara soal Pro'),
      href: OFFICIAL_LINKS.telegram || ROUTES.register,
    },
    {
      id: 'team',
      name: copy(language, 'Team', 'Team', 'Tim'),
      price: copy(language, '定制', 'Custom', 'Kustom'),
      period: copy(
        language,
        '私有化 / 年费',
        'Private deploy / annual',
        'Deploy privat / tahunan'
      ),
      blurb: copy(
        language,
        '飞书 / 钉钉 / 企微等亚太 IM、审计导出、SLA 与实施。适合小团队与工作室。',
        'Asia-Pacific IM (Feishu / DingTalk / WeCom), audit export, SLA and implementation — for small teams and studios.',
        'IM Asia-Pasifik (Feishu / DingTalk / WeCom), ekspor audit, SLA, dan implementasi.'
      ),
      highlights: [
        copy(language, 'Pro 能力基线 + 私有化部署', 'Pro baseline + private deployment', 'Dasar Pro + deploy privat'),
        copy(language, '企业 IM 触达与运维告警', 'Enterprise IM reach & ops alerts', 'Jangkauan IM enterprise & peringatan ops'),
        copy(language, '审计日志 / 资源健康面板', 'Audit logs / resource health hub', 'Log audit / hub kesehatan resource'),
        copy(language, '实施与年度维护', 'Implementation & annual maintenance', 'Implementasi & pemeliharaan tahunan'),
      ],
      cta: copy(language, '联系团队方案', 'Contact for Team', 'Hubungi untuk Tim'),
      href: OFFICIAL_LINKS.telegram || OFFICIAL_LINKS.twitter,
    },
  ]

  const principles = [
    {
      icon: ShieldCheck,
      title: copy(language, '软件本体可免费自托管', 'Software can be free to self-host', 'Perangkat lunak bisa gratis self-host'),
      body: copy(
        language,
        '开源 Community 不锁核心交易闭环。付费买的是托管省心、额度与支持——不是把密钥交给我们。',
        'Open-source Community does not lock the core trading loop. You pay for hosting convenience, limits and support — not for handing us your keys.',
        'Community open-source tidak mengunci loop trading inti. Anda membayar kenyamanan hosting, batas, dan dukungan.'
      ),
    },
    {
      icon: Zap,
      title: copy(language, 'AI 用量另行结算', 'AI usage billed separately', 'Pemakaian AI ditagih terpisah'),
      body: copy(
        language,
        'Claw402 / Base USDC 按次分析费与交易保证金分离；也可 BYOK 自带模型。',
        'Claw402 / Base USDC per-call analysis fees stay separate from trading margin. Or bring your own key (BYOK).',
        'Biaya analisis Claw402 / Base USDC terpisah dari margin. Atau BYOK.'
      ),
    },
    {
      icon: MessageSquare,
      title: copy(language, '社交触达 ≠ 聊天下单', 'IM reach ≠ chat-to-trade', 'Jangkauan IM ≠ chat-to-trade'),
      body: copy(
        language,
        'IM 负责告警与可选 Agent 对话；Web Chat 只读问行情。下单走 Trade / Autopilot。',
        'IM is for alerts and optional agents; Web Chat is read-only market Q&A. Orders go through Trade / Autopilot.',
        'IM untuk peringatan dan agen opsional; Web Chat baca-saja. Order lewat Trade / Autopilot.'
      ),
    },
  ]

  const featureRows: { name: string; community: string; cloud: string; pro: string; team: string }[] = [
    {
      name: copy(language, '自托管完整终端', 'Full self-hosted terminal', 'Terminal self-host penuh'),
      community: '✓',
      cloud: '✓',
      pro: '✓',
      team: '✓',
    },
    {
      name: copy(language, '官方托管运维', 'Official hosted ops', 'Ops hosted resmi'),
      community: '—',
      cloud: '✓',
      pro: '✓',
      team: '✓',
    },
    {
      name: copy(language, '多 Autopilot / 高额度', 'Multi-Autopilot / higher limits', 'Multi-Autopilot / batas tinggi'),
      community: copy(language, '自管', 'Self-managed', 'Mandiri'),
      cloud: copy(language, '标准', 'Standard', 'Standar'),
      pro: '✓',
      team: '✓',
    },
    {
      name: copy(language, '企业 IM（飞书/钉钉/企微）实施', 'Enterprise IM rollout', 'Rollout IM enterprise'),
      community: copy(language, '自配', 'DIY', 'DIY'),
      cloud: copy(language, '基础', 'Basic', 'Dasar'),
      pro: copy(language, '优先协助', 'Priority help', 'Bantuan prioritas'),
      team: '✓',
    },
    {
      name: copy(language, 'SLA / 审计导出', 'SLA / audit export', 'SLA / ekspor audit'),
      community: '—',
      cloud: '—',
      pro: copy(language, '可选', 'Optional', 'Opsional'),
      team: '✓',
    },
    {
      name: copy(language, 'AI 分析费', 'AI analysis fees', 'Biaya analisis AI'),
      community: 'Claw402 / BYOK',
      cloud: 'Claw402 / BYOK',
      pro: 'Claw402 / BYOK',
      team: 'Claw402 / BYOK',
    },
  ]

  return (
    <>
      <HeaderBar
        onLoginClick={() => setShowLoginModal(true)}
        isLoggedIn={isLoggedIn}
        language={language}
        onLanguageChange={setLanguage}
        user={user}
        onLogout={logout}
      />
      <div className="min-h-screen bg-strand-bg text-strand-text font-sans selection:bg-strand-gold selection:text-strand-bg">
        <main className="strand-pricing">
          <section className="pricing-hero">
            <span className="flow-eyebrow">
              <span />
              {copy(
                language,
                '开源获客 · 用量与托管变现',
                'Open source acquire · usage & hosting monetize',
                'Open source akuisisi · monetisasi usage & hosting'
              )}
            </span>
            <h1>
              {copy(
                language,
                '按你需要的控制感付费。',
                'Pay for the control you need.',
                'Bayar untuk kendali yang Anda butuhkan.'
              )}
            </h1>
            <p>
              {copy(
                language,
                'Strand 是非托管 AI 交易终端：Community 永远可自托管。Cloud / Pro 买的是省运维与额度；Team 面向亚太 IM 私有化。AI 分析费与交易保证金始终分开。',
                'Strand is a non-custodial AI trading terminal: Community stays free to self-host. Cloud / Pro buy less ops and higher limits; Team is private Asia-Pacific IM deploy. AI analysis fees always stay apart from trading margin.',
                'Strand adalah terminal trading AI non-kustodi: Community tetap gratis self-host. Cloud / Pro untuk lebih sedikit ops; Team untuk deploy IM privat.'
              )}
            </p>
            <div className="pricing-hero-actions">
              <Link to={ROUTES.autopilot} className="flow-primary">
                {copy(language, '免费自托管上手', 'Start free self-host', 'Mulai self-host gratis')}{' '}
                <ArrowRight size={18} />
              </Link>
              <Link to={ROUTES.help}>
                {copy(language, '看启动清单', 'Launch checklist', 'Checklist peluncuran')}{' '}
                <ArrowUpRight size={16} />
              </Link>
            </div>
          </section>

          <section className="pricing-tiers" aria-label="Plans">
            {tiers.map((tier) => (
              <article
                key={tier.id}
                className={`pricing-tier${tier.featured ? ' is-featured' : ''}`}
              >
                <header>
                  <div className="pricing-tier-top">
                    <h2>{tier.name}</h2>
                    {tier.badge ? <span className="pricing-badge">{tier.badge}</span> : null}
                  </div>
                  <div className="pricing-amount">
                    <strong>{tier.price}</strong>
                    <span>{tier.period}</span>
                  </div>
                  <p>{tier.blurb}</p>
                </header>
                <ul>
                  {tier.highlights.map((item) => (
                    <li key={item}>
                      <Check size={14} />
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
                {tier.note ? <p className="pricing-note">{tier.note}</p> : null}
                {tier.to ? (
                  <Link to={tier.to} className="pricing-cta">
                    {tier.cta} <ArrowRight size={16} />
                  </Link>
                ) : (
                  <a
                    href={tier.href}
                    target={tier.href?.startsWith('http') ? '_blank' : undefined}
                    rel={tier.href?.startsWith('http') ? 'noopener noreferrer' : undefined}
                    className="pricing-cta"
                  >
                    {tier.cta} <ArrowUpRight size={16} />
                  </a>
                )}
              </article>
            ))}
          </section>

          <section className="pricing-principles" aria-label="Pricing principles">
            {principles.map((item) => (
              <article key={item.title}>
                <item.icon size={22} />
                <h3>{item.title}</h3>
                <p>{item.body}</p>
              </article>
            ))}
          </section>

          <section className="pricing-compare" aria-label="Feature comparison">
            <div className="flow-section-heading">
              <span className="flow-eyebrow">
                {copy(language, '功能对照', 'Feature matrix', 'Matriks fitur')}
              </span>
              <h2>
                {copy(
                  language,
                  '选档位，不是买收益承诺。',
                  'Choose a tier — not a return promise.',
                  'Pilih paket — bukan janji return.'
                )}
              </h2>
              <p>
                {copy(
                  language,
                  'Strand 是软件工具。加密与杠杆交易有本金损失风险，任何档位都不保证盈利。',
                  'Strand is software. Crypto and leveraged trading can lose principal; no tier guarantees profit.',
                  'Strand adalah perangkat lunak. Trading berisiko kerugian; tidak ada paket yang menjamin profit.'
                )}
              </p>
            </div>
            <div className="pricing-table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>{copy(language, '能力', 'Capability', 'Kapabilitas')}</th>
                    <th>Community</th>
                    <th>Cloud</th>
                    <th>Pro</th>
                    <th>Team</th>
                  </tr>
                </thead>
                <tbody>
                  {featureRows.map((row) => (
                    <tr key={row.name}>
                      <th scope="row">{row.name}</th>
                      <td>{row.community}</td>
                      <td>{row.cloud}</td>
                      <td>{row.pro}</td>
                      <td>{row.team}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <section className="pricing-footer-cta">
            <div>
              <span className="flow-eyebrow">
                <Terminal size={14} />{' '}
                {copy(language, '现在就能用', 'Usable today', 'Bisa dipakai hari ini')}
              </span>
              <h2>
                {copy(
                  language,
                  '先跑 Community。',
                  'Start with Community.',
                  'Mulai dengan Community.'
                )}
                <br />
                {copy(
                  language,
                  '需要托管时再升级。',
                  'Upgrade when you want hosting.',
                  'Upgrade saat butuh hosting.'
                )}
              </h2>
              <p>
                {copy(
                  language,
                  '注册后按启动清单绑定 AI 与交易所；消息通道在 Settings → Resources / Messaging。',
                  'After sign-in, follow the launch checklist to bind AI and venues; messaging lives under Settings → Resources / Messaging.',
                  'Setelah masuk, ikuti checklist peluncuran; messaging di Settings → Resources / Messaging.'
                )}
              </p>
            </div>
            <div className="pricing-footer-actions">
              <Link to={ROUTES.register} className="flow-primary">
                {copy(language, '注册开始', 'Create account', 'Buat akun')}{' '}
                <ArrowRight size={18} />
              </Link>
              <Link to={ROUTES.home}>
                {copy(language, '返回首页', 'Back to home', 'Kembali ke beranda')}
              </Link>
            </div>
          </section>
        </main>

        <FooterSection language={language} />

        {showLoginModal && (
          <LoginModal
            onClose={() => setShowLoginModal(false)}
            language={language}
          />
        )}
      </div>
    </>
  )
}

export default PricingPage
