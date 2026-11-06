# Layouts del dashboard

## Resumen

El frontend soporta tres variantes de dashboard, seleccionables desde
`front/src/environments/environment.ts`. Todas comparten rutas, permisos y
componentes de negocio; solo cambia el shell de navegación.

| Variante     | Shell                                             |
| ------------ | ------------------------------------------------- |
| `sidebar`    | Sidebar vertical actual con flyouts               |
| `horizontal` | Tabs superiores + submenú izquierdo + flyouts     |
| `grid`       | Launcher de módulos con ventana maximizada        |

## Cambiar de variante

Editar:

```ts
DASHBOARD: {
  VARIANT: 'horizontal', // 'sidebar' | 'horizontal' | 'grid'
}
```

Guardar y dejar que `ng serve` recompile. No hay selector por usuario ni query
param: la variante es parte de la configuración del build.

## Variante horizontal

- Tabs superiores sin iconos, con etiqueta y caret si tienen hijos.
- Overflow con flechas de scroll y menú “Más” con todos los módulos.
- El tab activo muestra un submenú izquierdo.
- Los submenús anidados se abren como flyouts a la derecha.
- En móvil (`<= 900px`) los tabs y el submenú se ocultan y aparece una
  hamburguesa que abre un modal con navegación drill-down por niveles.
- Breadcrumb generado desde el menú activo.

## Variante grid

- La ruta `/dashboard` muestra una grilla plana con los módulos de primer nivel,
  sin agrupaciones por sección.
- Buscador local por nombre y descripción.
- Los módulos hoja navegan directo a su ruta.
- Los módulos con submenú (por ejemplo Tienda o Archivos) abren un modal de
  submenú; si el submenú tiene más niveles, el modal permite navegar hacia atrás.
- Al abrir un módulo se muestra una ventana maximizada con botón “Volver al
  inicio” y cierre con `Escape` (el botón también cierra).
- No hay multi-ventana ni taskbar en esta versión.

## Menú compartido

`front/src/app/services/menu.service.ts` centraliza:

- Filtro recursivo por permisos.
- Tab activo y trail para breadcrumbs.
- Primer nivel visible para la grilla.

`MenuItem` (`front/src/app/models/dashboard.ts`) admite:

```ts
description?: string;
group?: string;
badge?: string;
exact?: boolean;
submenu?: MenuItem[];
```

El menú actual incluye un ejemplo de tres niveles:

```text
Tienda
  Catálogo
    Categorías
    Productos
  Operaciones
    Ventas
```

## Comportamiento por permisos

- El sidebar, el horizontal y la grilla usan el mismo `MenuService`.
- Las rutas siguen protegidas por `permissionGuard`.
- Si un usuario no tiene permiso, el módulo no aparece y el acceso directo se
  redirige al dashboard con un aviso.
- Ejemplo: en Tienda, “Catálogo” requiere `shop`; “Operaciones → Ventas” requiere
  `ventas`. Un usuario con solo `shop` verá únicamente Catálogo. Esto es
  intencional: el menú refleja los permisos reales del backend.

## Archivos involucrados

| Capa       | Archivo                                                             |
| ---------- | ------------------------------------------------------------------- |
| Config     | `front/src/environments/environment.ts`                             |
| Tipos      | `front/src/app/models/dashboard.ts`                                 |
| Servicios  | `front/src/app/services/dashboard-config.service.ts`                |
| Servicios  | `front/src/app/services/menu.service.ts`                            |
| Shell      | `front/src/app/components/dashboard/layout/*`                       |
| Horizontal | `front/src/app/components/dashboard/layouts/horizontal-layout/*`    |
| Horizontal | `front/src/app/components/dashboard/horizontal-tabs/*`              |
| Horizontal | `front/src/app/components/dashboard/horizontal-subnav/*`            |
| Horizontal | `front/src/app/components/dashboard/nav-flyout/*`                   |
| Horizontal | `front/src/app/components/dashboard/mobile-drilldown/*`             |
| Grid       | `front/src/app/components/dashboard/layouts/grid-layout/*`          |
| Grid       | `front/src/app/components/dashboard/grid-home/*`                    |
| Grid       | `front/src/app/components/dashboard/shared/module-card/*`           |
| Compartido | `front/src/app/components/dashboard/shared/*`                       |

## Verificación

```powershell
cd front
npm run build
```

Prueba manual por variante:

1. `sidebar`: comportamiento actual.
2. `horizontal`: tabs, “Más”, scroll, submenú izquierdo, flyouts y móvil.
3. `grid`: grupos, buscador, tarjetas, ventana y “Volver al inicio”.
