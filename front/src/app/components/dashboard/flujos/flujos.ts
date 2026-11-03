import { Component, computed, inject, signal } from '@angular/core';
import { DomSanitizer, type SafeResourceUrl } from '@angular/platform-browser';

type FlujoKind = 'sequence' | 'workflow' | 'lifecycle' | 'dataflow';

interface FlujoDemo {
  slug: string;
  title: string;
  question: string;
  description: string;
  category: 'Acceso' | 'Datos' | 'Operaciones' | 'Reportes';
  type: FlujoKind;
  file: string;
}

const FLUJOS: FlujoDemo[] = [
  {
    slug: 'login',
    title: 'Acceso y restauración de sesión',
    question: '¿Cómo entra un usuario y cómo sobrevive la sesión?',
    description: 'Credenciales, bcrypt, JWT en cookie HttpOnly, guards y /auth/me.',
    category: 'Acceso',
    type: 'sequence',
    file: 'login.html',
  },
  {
    slug: 'logout',
    title: 'Cierre de sesión y revocación',
    question: '¿Qué pasa al salir y al revocar permisos?',
    description: 'Cierre manual, borrado de cookie, jti y sincronización de permisos.',
    category: 'Acceso',
    type: 'workflow',
    file: 'logout.html',
  },
  {
    slug: 'sesion-contador',
    title: 'Vida del contador de sesión',
    question: '¿Cómo cuenta la sesión y cuándo se cierra sola?',
    description: 'Aviso, crítico, expiración, renovación deslizante y auto-logout.',
    category: 'Acceso',
    type: 'lifecycle',
    file: 'sesion-contador.html',
  },
  {
    slug: 'usuarios-permisos',
    title: 'Usuarios, permisos y JWT',
    question: '¿Cómo se asignan permisos y cómo llegan al token?',
    description: 'Catálogo N:M, CRUD de usuarios, RemoveUserTokens y 403.',
    category: 'Acceso',
    type: 'workflow',
    file: 'usuarios-permisos.html',
  },
  {
    slug: 'mensajes-sse',
    title: 'Mensajes en vivo con SSE',
    question: '¿Cómo llega un cambio a todos los dashboards?',
    description: 'SSE global filtrado por permisos, snapshot por canal (module/name), reconexión y límites. Diagrama visual pendiente de actualizar.',
    category: 'Datos',
    type: 'sequence',
    file: 'mensajes-sse.html',
  },
  {
    slug: 'archivos',
    title: 'Archivos e imágenes: público y privado',
    question: '¿Dónde vive cada archivo y quién lo sirve?',
    description: 'Visibilidad, estado de la entidad, nginx, API y movimiento de archivos.',
    category: 'Datos',
    type: 'dataflow',
    file: 'archivos.html',
  },
  {
    slug: 'buses',
    title: 'Buses: listar, filtrar y mantener',
    question: '¿Cómo funciona el CRUD paginado de buses?',
    description: 'Count + página, filtros de servidor, crear, editar y eliminar.',
    category: 'Operaciones',
    type: 'workflow',
    file: 'buses.html',
  },
  {
    slug: 'tienda',
    title: 'Tienda: categorías, productos e imágenes',
    question: '¿Cómo se relacionan categorías, productos e imágenes?',
    description: 'CRUD relacionado, ImageManager, cascade y publicación por estado.',
    category: 'Operaciones',
    type: 'workflow',
    file: 'tienda.html',
  },
  {
    slug: 'reportes',
    title: 'Reportes y documentos',
    question: '¿Cómo se genera un reporte en PDF o Excel?',
    description: 'Datos actuales o generados, dos motores, plantillas y jobs.',
    category: 'Reportes',
    type: 'workflow',
    file: 'reportes.html',
  },
  {
    slug: 'documentos',
    title: 'Vida de un documento generado',
    question: '¿Qué estados y versiones tiene un documento?',
    description: 'Hash, correlativo, snapshot, regeneración y anulación.',
    category: 'Reportes',
    type: 'lifecycle',
    file: 'documentos.html',
  },
  {
    slug: 'instalaciones-mapas',
    title: 'Instalaciones en el mapa',
    question: '¿Cómo se gestionan instalaciones desde el mapa?',
    description: 'Carga completa, filtros en cliente, marcadores y CRUD.',
    category: 'Operaciones',
    type: 'workflow',
    file: 'instalaciones-mapas.html',
  },
];

const CATEGORIES = ['Todos', 'Acceso', 'Datos', 'Operaciones', 'Reportes'] as const;
type Category = (typeof CATEGORIES)[number];

const TYPE_LABEL: Record<FlujoKind, string> = {
  sequence: 'Secuencia',
  workflow: 'Flujo',
  lifecycle: 'Ciclo de vida',
  dataflow: 'Datos',
};

@Component({
  selector: 'app-flujos',
  templateUrl: './flujos.html',
  styleUrl: './flujos.css',
})
export class FlujosComponent {
  private readonly sanitizer = inject(DomSanitizer);

  readonly categories = CATEGORIES;
  readonly flujos = FLUJOS;
  readonly category = signal<Category>('Todos');
  readonly selected = signal<FlujoDemo>(FLUJOS[0]);

  readonly filtered = computed(() => {
    const category = this.category();
    return category === 'Todos'
      ? this.flujos
      : this.flujos.filter((flujo) => flujo.category === category);
  });

  readonly previewUrl = computed<SafeResourceUrl>(() =>
    this.sanitizer.bypassSecurityTrustResourceUrl(this.assetUrl(this.selected().file)),
  );

  typeLabel(type: FlujoKind): string {
    return TYPE_LABEL[type];
  }

  setCategory(category: Category): void {
    this.category.set(category);
    const list = this.filtered();
    if (!list.some((flujo) => flujo.slug === this.selected().slug)) {
      this.selected.set(list[0] ?? this.flujos[0]);
    }
  }

  select(flujo: FlujoDemo): void {
    this.selected.set(flujo);
  }

  openFull(file: string): void {
    window.open(this.assetUrl(file), '_blank', 'noopener');
  }

  private assetUrl(file: string): string {
    return new URL(`flujos/${file}`, document.baseURI).toString();
  }
}
