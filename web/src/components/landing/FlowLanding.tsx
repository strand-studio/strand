import {
  ArrowRight,
  ArrowUpRight,
  BarChart3,
  Bell,
  Layers3,
  MessageSquare,
  ShieldCheck,
  SlidersHorizontal,
  Terminal,
  Trophy,
  Wallet,
  Check,
} from 'lucide-react'
import { Link } from 'react-router'
import { ROUTES } from '../../router/paths'
import { OFFICIAL_LINKS } from '../../constants/branding'
import { BrandLogo } from '../common/BrandLogo'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'

function copy(lang: string, zh: string, en: string, id?: string) {
  if (lang === 'zh') return zh
  if (lang === 'id' && id) return id
  return en
}

const VENUES = [
  'Hyperliquid',
  'Binance',
  'Bybit',
  'OKX',
  'Gate',
  'Bitget',
  'KuCoin',
  'Lighter',
  'Aster',
] as const

export default function FlowLanding() {
  const { language } = useLanguage()

  const features = [
    {
      icon: BarChart3,
      title: copy(
        language,
        '先看清市场',
        'Read the board first',
        'Baca papan dulu'
      ),
      body: copy(
        language,
        'Markets 汇总 Hyperliquid 与主流 CEX 永续行情；点进标的看 K 线与同所热门——无需登录即可浏览。',
        'Markets aggregates Hyperliquid and major CEX perps. Open a symbol for candles and related markets — browse without signing in.',
        'Markets merangkum Hyperliquid dan CEX utama. Buka simbol untuk candle dan pasar terkait — tanpa login.'
      ),
      label: t('marketsNav', language),
      to: ROUTES.markets,
    },
    {
      icon: Layers3,
      title: copy(
        language,
        '确认后再下单',
        'Approve before you send',
        'Setujui sebelum kirim'
      ),
      body: copy(
        language,
        'Trade 对比市价 / 限价 / Ladder / TWAP，看清入场与退出再确认；Markets / Portfolio 可 Chart-to-order。',
        'Trade compares market / limit / Ladder / TWAP before you confirm. Chart-to-order from Markets or Portfolio when you want the final say.',
        'Trade membandingkan market / limit / Ladder / TWAP sebelum konfirmasi. Chart-to-order dari Markets atau Portfolio.'
      ),
      label: t('tradeNav', language),
      to: ROUTES.planning,
    },
    {
      icon: SlidersHorizontal,
      title: copy(
        language,
        '或交给自动驾驶',
        'Or let Autopilot run',
        'Atau biarkan Autopilot'
      ),
      body: copy(
        language,
        '绑定 Claw402 / BYOK 与交易所，Preflight 绿灯后再启动。方向板驱动，日亏 / 回撤硬闸在模型之外。',
        'Bind Claw402 / BYOK and your venue, pass Preflight, then launch. The board drives; daily-loss and drawdown gates sit outside the model.',
        'Hubungkan Claw402 / BYOK dan bursa, lulus Preflight, lalu jalankan. Papan mengarahkan; hard gate di luar model.'
      ),
      label: t('autopilotNav', language),
      to: ROUTES.autopilot,
    },
    {
      icon: Bell,
      title: copy(
        language,
        '状态跟到你的 IM',
        'Alerts meet you in chat',
        'Peringatan di chat Anda'
      ),
      body: copy(
        language,
        'Telegram / Discord / Slack / 飞书 / 钉钉 / 微信等可收硬闸与崩溃告警；可选绑对话 Agent。交易仍在本机执行——不是聊天下单。',
        'Telegram, Discord, Slack, Feishu, DingTalk, WeChat and more for hard-gate and crash alerts; optional conversation agents. Execution stays on your machine — not chat-to-trade.',
        'Telegram, Discord, Slack, Feishu, DingTalk, WeChat untuk peringatan hard gate; agen percakapan opsional. Eksekusi tetap di mesin Anda.'
      ),
      label: t('pricingNav', language),
      to: ROUTES.pricing,
    },
    {
      icon: Wallet,
      title: copy(
        language,
        '仓位、费用与规则一处看清',
        'Positions, fees, and rules',
        'Posisi, biaya, dan aturan'
      ),
      body: copy(
        language,
        'Portfolio 看权益与决策；Strategy 管杠杆与日亏硬闸；AI 手续费钱包（Base USDC）与交易保证金分开。',
        'Portfolio tracks equity and decisions; Strategy owns leverage and daily-loss gates. AI fee wallet (Base USDC) stays apart from trading margin.',
        'Portfolio melacak ekuitas dan keputusan; Strategy mengatur leverage dan batas rugi. Dompet biaya AI terpisah dari margin.'
      ),
      label: t('portfolioNav', language),
      to: ROUTES.portfolio,
    },
    {
      icon: ShieldCheck,
      title: copy(
        language,
        '启动前先过 Preflight',
        'Preflight before you go live',
        'Preflight sebelum live'
      ),
      body: copy(
        language,
        'AI 费钱包、交易所余额、交易授权未就绪则拒绝启动；Safe Mode 可只平仓不开新仓。',
        'Start is refused until the AI fee wallet, exchange balance, and trade auth are ready. Safe Mode can close-only when gates trip.',
        'Start ditolak sampai dompet biaya AI, saldo, dan otorisasi siap. Safe Mode bisa close-only saat hard gate aktif.'
      ),
      label: t('helpNav', language),
      to: ROUTES.help,
    },
  ]

  const steps = [
    {
      n: '01',
      title: copy(language, '浏览行情', 'Browse markets', 'Jelajahi pasar'),
      body: copy(
        language,
        '打开 Markets，切换交易所，进入标的详情。',
        'Open Markets, switch venues, open a symbol detail.',
        'Buka Markets, ganti venue, buka detail simbol.'
      ),
    },
    {
      n: '02',
      title: copy(language, '选路径', 'Choose a path', 'Pilih jalur'),
      body: copy(
        language,
        'Trade 人工确认，或 Autopilot 跟方向板自动执行。',
        'Trade with human approval, or Autopilot on the direction board.',
        'Trade dengan persetujuan manusia, atau Autopilot mengikuti papan arah.'
      ),
    },
    {
      n: '03',
      title: copy(language, '守住风控与触达', 'Keep gates & reach', 'Jaga gate & jangkauan'),
      body: copy(
        language,
        '设硬闸、盯 Portfolio，把告警绑到日常 IM；密钥始终在本机。',
        'Set hard gates, watch Portfolio, bind alerts to the apps you already use — keys stay on your machine.',
        'Atur hard gate, pantau Portfolio, ikat peringatan ke IM Anda — kunci tetap lokal.'
      ),
    },
  ]

  return (
    <div className="strand-flow-landing">
      <section className="flow-hero">
        <div className="flow-hero-copy">
          <p className="flow-brand" aria-label="Strand">
            Strand
          </p>
          <span className="flow-eyebrow">
            <span />
            {copy(
              language,
              '开源 · 自托管 · 非托管资金',
              'Open source · Self-hosted · Non-custodial',
              'Open source · Self-hosted · Non-kustodi'
            )}
          </span>
          <h1>
            {copy(
              language,
              '会交易的智能，',
              'Intelligence that can trade.',
              'Kecerdasan yang bisa trading.'
            )}
            <br />
            <em>
              {copy(
                language,
                '控制权在你手里。',
                'Control that stays yours.',
                'Kendali tetap milik Anda.'
              )}
            </em>
          </h1>
          <p>
            {copy(
              language,
              'Strand 是跑在你机器上的 AI 交易终端：Autopilot 跟随 Claw402 方向板，Trade 让你先规划再下单；硬风控在模型之外，告警可跟到 Telegram / 飞书 / 钉钉等——密钥与资金不离开本机。',
              'Strand is the AI trading terminal on your machine: Autopilot follows the Claw402 board, Trade lets you plan before you send. Hard gates sit outside the model; alerts can meet you in Telegram, Feishu, DingTalk and more — keys and funds never leave your server.',
              'Strand adalah terminal trading AI di mesin Anda: Autopilot mengikuti papan Claw402, Trade meninjau rencana sebelum kirim. Hard gate di luar model; peringatan bisa ke Telegram/Feishu — kunci tidak meninggalkan server Anda.'
            )}
          </p>
          <div className="flow-hero-actions">
            <Link to={ROUTES.markets} className="flow-primary">
              {copy(language, '打开 Markets', 'Open Markets', 'Buka Markets')}{' '}
              <ArrowRight size={18} />
            </Link>
            <Link to={ROUTES.pricing}>
              {copy(language, '查看定价', 'See pricing', 'Lihat harga')}{' '}
              <ArrowUpRight size={16} />
            </Link>
          </div>
          <div className="flow-hero-trust">
            <span>
              <Check size={14} />{' '}
              {copy(language, '本机运行', 'Runs on your machine', 'Berjalan di mesin Anda')}
            </span>
            <span>
              <Check size={14} />{' '}
              {copy(
                language,
                'AI 费与保证金分离',
                'AI fees ≠ trading margin',
                'Biaya AI ≠ margin trading'
              )}
            </span>
            <span>
              <Check size={14} />{' '}
              {copy(
                language,
                '告警跟到 IM',
                'Alerts in your IM',
                'Peringatan di IM'
              )}
            </span>
          </div>
        </div>

        <div
          className="flow-product-preview"
          aria-label={copy(
            language,
            '控制台工作区示意',
            'Console workspace preview',
            'Pratinjau ruang kerja konsol'
          )}
        >
          <div className="flow-preview-header">
            <BrandLogo />
            <span>{t('marketsNav', language)}</span>
            <small>
              {copy(language, '工作区预览', 'Workspace preview', 'Pratinjau')}
            </small>
          </div>
          <div className="flow-preview-content">
            <div className="flow-preview-market">
              <span>
                {copy(
                  language,
                  '多所永续 · 方向板 · 硬风控',
                  'Multi-venue perps · board · hard gates',
                  'Perp multi-venue · papan · hard gate'
                )}
              </span>
              <b>
                {copy(
                  language,
                  '从行情到执行，状态跟到 IM。',
                  'From board to execution — status follows you.',
                  'Dari papan ke eksekusi — status mengikuti Anda.'
                )}
              </b>
            </div>
            <div className="flow-preview-board" aria-hidden>
              {[
                ['BTC', '+1.24%', '↑'],
                ['ETH', '-0.62%', '↓'],
                ['SOL', '+2.18%', '↑'],
                ['XRP', '+0.41%', '↑'],
              ].map(([sym, chg, dir]) => (
                <div key={sym} data-dir={dir === '↑' ? 'up' : 'down'}>
                  <strong>{sym}</strong>
                  <span>{chg}</span>
                </div>
              ))}
            </div>
            <svg
              className="flow-preview-chart"
              viewBox="0 0 600 180"
              role="img"
              aria-label={copy(
                language,
                '示意价格曲线',
                'Illustrative price curve',
                'Kurva harga ilustratif'
              )}
            >
              <defs>
                <linearGradient
                  id="flow-chart-fill"
                  x1="0"
                  y1="0"
                  x2="0"
                  y2="1"
                >
                  <stop stopColor="#295be8" stopOpacity=".13" />
                  <stop offset="1" stopColor="#295be8" stopOpacity="0" />
                </linearGradient>
              </defs>
              {[40, 80, 120].map((y) => (
                <path key={y} d={`M0 ${y}H600`} stroke="var(--panel-border)" />
              ))}
              <path
                d="M0 140 40 128 70 134 110 112 150 118 190 92 230 98 270 74 310 86 360 58 410 70 460 48 510 56 560 42V180H0Z"
                fill="url(#flow-chart-fill)"
              />
              <path
                d="m0 140 40-12 30 6 40-22 40 6 40-26 40 6 40-24 40 12 50-28 50 12 50-22 50 8 50-14"
                fill="none"
                stroke="#295be8"
                strokeWidth="2.5"
                strokeLinejoin="round"
              />
            </svg>
            <div className="flow-preview-plans">
              {[
                {
                  title: t('tradeNav', language),
                  tip: copy(language, '对比计划', 'Compare plans', 'Bandingkan'),
                },
                {
                  title: t('autopilotNav', language),
                  tip: copy(language, '方向板驱动', 'Board-driven', 'Ikuti papan'),
                },
                {
                  title: copy(language, '消息通道', 'Messaging', 'Pesan'),
                  tip: copy(language, '告警跟到 IM', 'IM alerts', 'Peringatan IM'),
                },
                {
                  title: t('leaderboardNav', language),
                  tip: copy(language, '公开排名', 'Public ranks', 'Peringkat'),
                },
              ].map((item, i) => (
                <div key={item.title} data-selected={i === 0}>
                  <span>0{i + 1}</span>
                  <strong>{item.title}</strong>
                  <small>{item.tip}</small>
                </div>
              ))}
            </div>
            <div className="flow-preview-bottom">
              <span>
                <Check size={14} />{' '}
                {copy(language, '密钥留在本机', 'Keys stay local', 'Kunci tetap lokal')}
              </span>
              <Link to={ROUTES.autopilot}>
                {copy(language, '设置 Autopilot', 'Set up Autopilot', 'Siapkan Autopilot')}{' '}
                <ArrowUpRight size={14} />
              </Link>
            </div>
          </div>
        </div>
      </section>

      <section
        className="flow-exchanges"
        aria-label={copy(
          language,
          '支持的交易场所',
          'Supported trading venues',
          'Venue trading yang didukung'
        )}
      >
        <span>
          {copy(
            language,
            '你的市场，连上即可。',
            'Your markets. Connected.',
            'Pasar Anda. Terhubung.'
          )}
        </span>
        {VENUES.map((name) => (
          <b key={name}>{name}</b>
        ))}
      </section>

      <section className="flow-steps" aria-label="How Strand works">
        <div className="flow-section-heading">
          <span className="flow-eyebrow">
            {copy(language, '怎么用', 'How it works', 'Cara kerja')}
          </span>
          <h2>{copy(language, '三步上手', 'Three steps', 'Tiga langkah')}</h2>
          <p>
            {copy(
              language,
              '从公开行情开始，再选人工交易或自动驾驶，并把告警绑到 IM。',
              'Start from the public board, pick Trade or Autopilot, then bind alerts to your IM.',
              'Mulai dari papan publik, pilih Trade atau Autopilot, lalu ikat peringatan ke IM.'
            )}
          </p>
        </div>
        {steps.map((step) => (
          <article key={step.n} className="flow-step">
            <span>{step.n}</span>
            <h3>{step.title}</h3>
            <p>{step.body}</p>
          </article>
        ))}
      </section>

      <section className="flow-features">
        <div className="flow-section-heading">
          <span className="flow-eyebrow">
            {copy(language, '一个控制台', 'One console', 'Satu konsol')}
          </span>
          <h2>
            {copy(language, '少切换。', 'Less switching.', 'Lebih sedikit berpindah.')}
            <br />
            {copy(language, '多一点清楚。', 'More clarity.', 'Lebih jelas.')}
          </h2>
          <p>
            {copy(
              language,
              '行情、计划、自动驾驶、风控与消息触达在同一套导航里——按意图选入口即可。',
              'Markets, plans, Autopilot, risk and messaging live in one nav — pick the door that matches your intent.',
              'Markets, rencana, Autopilot, risiko, dan pesan dalam satu navigasi.'
            )}
          </p>
        </div>
        {features.map((item) => (
          <article key={item.title}>
            <item.icon size={23} />
            <h3>{item.title}</h3>
            <p>{item.body}</p>
            <Link to={item.to}>
              {item.label}
              <ArrowUpRight size={16} />
            </Link>
          </article>
        ))}
      </section>

      <section className="flow-paths" aria-label="Quick paths">
        <Link to={ROUTES.pricing} className="flow-path">
          <Wallet size={18} />
          <div>
            <strong>{t('pricingNav', language)}</strong>
            <span>
              {copy(
                language,
                '开源免费 · Cloud / Pro / 团队分档',
                'Free self-host · Cloud / Pro / Team tiers',
                'Self-host gratis · Cloud / Pro / Tim'
              )}
            </span>
          </div>
          <ArrowUpRight size={16} />
        </Link>
        <Link to={ROUTES.leaderboard} className="flow-path">
          <Trophy size={18} />
          <div>
            <strong>{t('leaderboardNav', language)}</strong>
            <span>
              {copy(
                language,
                '公开浏览 opt-in 排名',
                'Browse opt-in rankings',
                'Lihat peringkat opt-in'
              )}
            </span>
          </div>
          <ArrowUpRight size={16} />
        </Link>
        <Link to={ROUTES.chat} className="flow-path">
          <MessageSquare size={18} />
          <div>
            <strong>{t('chatNav', language)}</strong>
            <span>
              {copy(
                language,
                '只读问行情（需登录，不下单）',
                'Read-only market Q&A (sign-in, no orders)',
                'Tanya pasar baca-saja (perlu masuk, tanpa order)'
              )}
            </span>
          </div>
          <ArrowUpRight size={16} />
        </Link>
        <Link to={ROUTES.help} className="flow-path">
          <Terminal size={18} />
          <div>
            <strong>{t('helpNav', language)}</strong>
            <span>
              {copy(
                language,
                '钱包、启动与自托管清单',
                'Wallets, launch & self-host checklist',
                'Dompet, peluncuran & checklist host'
              )}
            </span>
          </div>
          <ArrowUpRight size={16} />
        </Link>
      </section>

      <section className="flow-local">
        <div>
          <span className="flow-eyebrow">
            <Terminal size={14} />{' '}
            {copy(
              language,
              '为本地运行而建',
              'Built to run locally',
              'Dirancang untuk lokal'
            )}
          </span>
          <h2>
            {copy(language, '你的工作区。', 'Your workspace.', 'Ruang kerja Anda.')}
            <br />
            {copy(language, '按你的规则。', 'On your terms.', 'Dengan aturan Anda.')}
          </h2>
          <p>
            {copy(
              language,
              '在本机跑 Go API 与 Web 控制台，连接你的账户与消息通道，再选 Trade 或 Autopilot——密钥不离开你的服务器。不想自己运维？看 Pricing 的 Cloud / 团队档。',
              'Run the Go API and web console on your machine, connect accounts and messaging, then choose Trade or Autopilot — keys never leave your server. Prefer not to operate it yourself? See Cloud and Team on Pricing.',
              'Jalankan Go API dan konsol web di mesin Anda, hubungkan akun dan pesan, lalu pilih Trade atau Autopilot. Tidak ingin mengelola sendiri? Lihat Cloud dan Tim di Pricing.'
            )}
          </p>
        </div>
        <div className="flow-local-actions">
          {OFFICIAL_LINKS.github ? (
            <a
              href={OFFICIAL_LINKS.github}
              target="_blank"
              rel="noopener noreferrer"
            >
              {copy(language, '查看源码', 'View source', 'Lihat sumber')}{' '}
              <ArrowUpRight size={18} />
            </a>
          ) : null}
          <Link to={ROUTES.pricing}>
            {copy(language, '对比方案', 'Compare plans', 'Bandingkan paket')}{' '}
            <ArrowUpRight size={18} />
          </Link>
        </div>
      </section>
    </div>
  )
}
