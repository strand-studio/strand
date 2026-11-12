import {
  ArrowRight,
  ArrowUpRight,
  BarChart3,
  Layers3,
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
        'Markets 汇总 Hyperliquid 与主流 CEX 永续行情；点进标的详情看 K 线、换算与同所热门——无需登录即可浏览。',
        'Markets aggregates Hyperliquid and major CEX perps. Open a symbol for candles, converter and related markets — browse without signing in.',
        'Markets merangkum Hyperliquid dan CEX utama. Buka simbol untuk candle, konverter, dan pasar terkait — tanpa login.'
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
        'Trade 对比市价 / 限价 / Ladder / TWAP，看清入场、仓位与退出，再人工确认——适合你想自己拍板的时候。',
        'Trade compares market / limit / Ladder / TWAP so you see entry, size and exits before you confirm — when you want the final say.',
        'Trade membandingkan market / limit / Ladder / TWAP agar entry, size, dan exit jelas sebelum Anda konfirmasi.'
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
        '绑定 Claw402 / BYOK 与交易所，在 Strategy 设硬风控，再启动 Autopilot。方向板驱动，硬闸在模型之外。',
        'Bind Claw402 / BYOK and your venue, set hard gates on Strategy, then launch Autopilot. The board drives; hard gates sit outside the model.',
        'Hubungkan Claw402 / BYOK dan bursa, atur hard gate di Strategy, lalu jalankan Autopilot. Papan mengarahkan; hard gate di luar model.'
      ),
      label: t('autopilotNav', language),
      to: ROUTES.autopilot,
    },
    {
      icon: Wallet,
      title: copy(
        language,
        '仓位与规则一处看清',
        'Positions and rules in view',
        'Posisi dan aturan terlihat'
      ),
      body: copy(
        language,
        'Portfolio 看权益与决策；Strategy 管杠杆与日亏硬闸；Leaderboard 公开对比 opt-in 实例。AI 手续费钱包与交易保证金分开。',
        'Portfolio tracks equity and decisions; Strategy owns leverage and daily-loss gates; Leaderboard compares opt-in runners. AI fee wallet stays apart from trading margin.',
        'Portfolio melacak ekuitas dan keputusan; Strategy mengatur leverage dan batas rugi harian; Leaderboard membandingkan runner opt-in.'
      ),
      label: t('portfolioNav', language),
      to: ROUTES.portfolio,
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
      title: copy(
        language,
        '选路径',
        'Choose a path',
        'Pilih jalur'
      ),
      body: copy(
        language,
        'Trade 人工确认，或 Autopilot 跟方向板自动执行。',
        'Trade with human approval, or Autopilot on the direction board.',
        'Trade dengan persetujuan manusia, atau Autopilot mengikuti papan arah.'
      ),
    },
    {
      n: '03',
      title: copy(
        language,
        '守住风控',
        'Keep the gates',
        'Jaga hard gate'
      ),
      body: copy(
        language,
        '在 Strategy 设限，在 Portfolio 盯仓，密钥始终在本机。',
        'Set limits on Strategy, watch Portfolio — keys stay on your machine.',
        'Atur batas di Strategy, pantau Portfolio — kunci tetap di mesin Anda.'
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
              '开源 · 自托管 · 不托管资金',
              'Open source · Self-hosted · Non-custodial',
              'Open source · Self-hosted · Non-kustodi'
            )}
          </span>
          <h1>
            {copy(language, '读懂市场。', 'Read the market.', 'Baca pasar.')}
            <br />
            <em>
              {copy(
                language,
                '再决定怎么交易。',
                'Then decide how to trade.',
                'Lalu putuskan cara trade.'
              )}
            </em>
          </h1>
          <p>
            {copy(
              language,
              'Strand 是跑在你机器上的 AI 交易终端：Markets 看行情，Trade 审计划，Autopilot 跟 Claw402 方向板执行——硬风控在模型之外，密钥不离开本机。',
              'Strand is the AI trading terminal on your machine: Markets for the board, Trade to review plans, Autopilot to follow the Claw402 direction board — hard risk gates outside the model, keys never leave your server.',
              'Strand adalah terminal trading AI di mesin Anda: Markets untuk papan, Trade untuk meninjau rencana, Autopilot mengikuti papan Claw402 — hard gate di luar model, kunci tidak meninggalkan server Anda.'
            )}
          </p>
          <div className="flow-hero-actions">
            <Link to={ROUTES.markets} className="flow-primary">
              {copy(
                language,
                '打开 Markets',
                'Open Markets',
                'Buka Markets'
              )}{' '}
              <ArrowRight size={18} />
            </Link>
            <Link to={ROUTES.autopilot}>
              {copy(
                language,
                '设置 Autopilot',
                'Set up Autopilot',
                'Siapkan Autopilot'
              )}{' '}
              <ArrowUpRight size={16} />
            </Link>
          </div>
          <div className="flow-hero-trust">
            <span>
              <Check size={14} />{' '}
              {copy(
                language,
                '本机运行',
                'Runs on your machine',
                'Berjalan di mesin Anda'
              )}
            </span>
            <span>
              <Check size={14} />{' '}
              {copy(
                language,
                '你的交易所账户',
                'Your exchange accounts',
                'Akun bursa Anda'
              )}
            </span>
            <span>
              <Check size={14} />{' '}
              {copy(
                language,
                '行情可匿名浏览',
                'Markets without login',
                'Markets tanpa login'
              )}
            </span>
          </div>
        </div>

        <div
          className="flow-product-preview"
          aria-label={copy(
            language,
            'Markets 工作区示意',
            'Markets workspace preview',
            'Pratinjau ruang kerja Markets'
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
                  '多所永续 · 实时板',
                  'Multi-venue perps · live board',
                  'Perp multi-venue · papan live'
                )}
              </span>
              <b>
                {copy(
                  language,
                  '从行情到计划，一条路径。',
                  'From board to plan, one path.',
                  'Dari papan ke rencana, satu jalur.'
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
                <path key={y} d={`M0 ${y}H600`} stroke="#edf1f7" />
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
                  title: t('leaderboardNav', language),
                  tip: copy(language, '公开排名', 'Public ranks', 'Peringkat publik'),
                },
                {
                  title: t('helpNav', language),
                  tip: copy(language, '自托管指南', 'Self-host guide', 'Panduan host'),
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
                {copy(
                  language,
                  '密钥留在本机',
                  'Keys stay local',
                  'Kunci tetap lokal'
                )}
              </span>
              <Link to={ROUTES.markets}>
                {copy(
                  language,
                  '进入 Markets',
                  'Enter Markets',
                  'Masuk Markets'
                )}{' '}
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
          <h2>
            {copy(language, '三步上手', 'Three steps', 'Tiga langkah')}
          </h2>
          <p>
            {copy(
              language,
              '从公开行情开始，再选人工交易或自动驾驶。',
              'Start from the public board, then pick human Trade or Autopilot.',
              'Mulai dari papan publik, lalu pilih Trade manual atau Autopilot.'
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
            {copy(
              language,
              '一个控制台',
              'One console',
              'Satu konsol'
            )}
          </span>
          <h2>
            {copy(
              language,
              '少切换。',
              'Less switching.',
              'Lebih sedikit berpindah.'
            )}
            <br />
            {copy(
              language,
              '多一点清楚。',
              'More clarity.',
              'Lebih jelas.'
            )}
          </h2>
          <p>
            {copy(
              language,
              '行情、计划、自动驾驶与风控在同一套导航里——按意图选入口即可。',
              'Markets, plans, Autopilot and risk live in one nav — pick the door that matches your intent.',
              'Markets, rencana, Autopilot, dan risiko dalam satu navigasi — pilih pintu sesuai niat Anda.'
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
        <Link to={ROUTES.chat} className="flow-path">
          <BarChart3 size={18} />
          <div>
            <strong>{t('chatNav', language)}</strong>
            <span>
              {copy(
                language,
                '只读工具问行情（需登录）',
                'Read-only market tools (sign-in)',
                'Alat pasar baca-saja (perlu masuk)'
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
            {copy(
              language,
              '你的工作区。',
              'Your workspace.',
              'Ruang kerja Anda.'
            )}
            <br />
            {copy(
              language,
              '按你的规则。',
              'On your terms.',
              'Dengan aturan Anda.'
            )}
          </h2>
          <p>
            {copy(
              language,
              '在本机跑 Go API 与 Web 控制台，连接你的账户，再选 Trade 或 Autopilot——密钥不离开你的服务器。',
              'Run the Go API and web console on your machine, connect your accounts, then choose Trade or Autopilot — keys never leave your server.',
              'Jalankan Go API dan konsol web di mesin Anda, hubungkan akun, lalu pilih Trade atau Autopilot — kunci tidak meninggalkan server Anda.'
            )}
          </p>
        </div>
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
      </section>
    </div>
  )
}
