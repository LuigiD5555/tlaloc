# COMPENDIO MAESTRO DE ORIGAMI

## Visión, ciencia, arquitectura, evolución y objetivos

### Estado conceptual: Origami 6.x
### Documento vivo

---

# 1. ¿QUÉ ES ORIGAMI?

**Origami es un lenguaje y una máquina de representación computacional para describir estado, relaciones, estructura, transformación, dinámica, observación y percepción.**

Su idea central no es almacenar información simplemente como texto, bytes o imágenes.

Origami intenta representar:

- qué cosas existen;
- en qué estado se encuentran;
- cómo se relacionan;
- qué depende de qué;
- qué está presente;
- qué está ausente;
- qué permanece indeterminado;
- qué está latente;
- qué puede cambiar;
- qué reglas producen esos cambios;
- qué ocurre a lo largo del tiempo;
- qué estructuras emergen de las interacciones;
- qué puede observar un determinado observador;
- qué parte de la información necesita desplegarse para resolver una consulta;
- y qué evidencia permite considerar una respuesta válida.

Una imagen puede ser una representación de una máquina Origami.

Pero:

**Origami no es una imagen.**

Formalmente:

\[
\text{STATE} \neq \text{VISUAL PROJECTION}
\]

La imagen es solamente uno de los posibles canales mediante los cuales un estado Origami puede manifestarse.

---

# 2. LA IDEA FUNDAMENTAL

Una computadora tradicional tiende a representar información mediante estructuras explícitas:

```text
dato
dato
dato
dato
```

Origami busca representar también las **reglas y relaciones capaces de producir esos datos**.

En lugar de guardar:

```text
AAAAAAAABBBBBBBBCCCCCCCC
```

podría representar algo conceptualmente semejante a:

```text
REPEAT(A,8)
REPEAT(B,8)
REPEAT(C,8)
```

Pero Origami quiere llevar esta idea mucho más lejos.

Puede representar:

```text
A depende de B
B inhibe C
C activa D
D modifica A
```

y permitir que esas relaciones evolucionen.

Por ello, una representación Origami puede actuar simultáneamente como:

- estructura de datos;
- grafo;
- sistema de reglas;
- máquina de estados;
- representación generativa;
- sistema dinámico;
- memoria direccionable;
- estructura perceptual;
- programa restringido;
- soporte para experimentos;
- fuente consultable.

---

# 3. EL PRINCIPIO DEL ORIGAMI

El nombre resume bastante bien la filosofía.

Una hoja de papel contiene una estructura relativamente simple.

Mediante reglas de plegado puede producir una configuración mucho más compleja.

Origami intenta hacer algo equivalente con información.

## FOLD

Comprimir o representar una estructura mediante:

- relaciones;
- reglas;
- patrones;
- referencias;
- transformaciones;
- jerarquías;
- simetrías;
- predicciones;
- residuos.

## UNFOLD

Reconstruir o materializar únicamente aquello que resulta necesario.

La intención no es necesariamente:

```text
comprimir → descomprimir todo
```

sino:

```text
representar
      ↓
preguntar
      ↓
localizar
      ↓
desplegar únicamente dependencias necesarias
      ↓
resolver
```

Éste es el origen del concepto de **Selective Unfolding**.

---

# 4. DE DÓNDE NACIÓ ORIGAMI

Origami no apareció directamente en su forma actual.

Ha pasado por varias etapas.

---

# 5. PRIMERA ETAPA — REPRESENTACIÓN COMPACTA

La primera intuición fue estudiar si grandes cantidades de información podían representarse mediante estructuras mucho menores explotando regularidades.

Aparecieron conceptos como:

- cápsulas;
- folds;
- backlinks;
- referencias;
- estructuras reutilizables;
- recuperación parcial;
- reconstrucción condicionada por presupuesto.

La idea inicial estaba todavía muy cerca del problema de compresión.

Pero apareció una observación crucial:

> almacenar información literalmente no es siempre la mejor manera de representar información.

Si un conjunto posee estructura, puede resultar más eficiente almacenar:

\[
\text{modelo} + \text{parámetros} + \text{excepciones}
\]

que almacenar todos sus elementos individualmente.

Éste acabaría siendo uno de los fundamentos de HyperFold.

---

# 6. SEGUNDA ETAPA — VCL

El siguiente paso fue el **Visual Context Language**.

Aquí comenzamos a investigar si una imagen podía funcionar como un lenguaje estructurado que un modelo multimodal pudiera interpretar.

Las primeras versiones probaron:

- posiciones;
- colores;
- formas;
- orientación;
- regiones;
- patrones;
- ruido;
- escala;
- superglyphs;
- relaciones visuales.

VCL evolucionó hasta manejar decenas de dimensiones visuales potenciales.

Pero apareció un problema fundamental.

## Capacidad matemática no significa capacidad perceptual

Supongamos que tenemos:

- 8 colores;
- 8 formas;
- 8 orientaciones;
- 8 tamaños.

Matemáticamente podríamos pensar:

\[
8^4 = 4096
\]

configuraciones.

Pero eso no significa que un observador pueda distinguir de manera fiable las 4096.

Al reducir la imagen, comprimirla, cambiar la resolución o mostrarla a un VLM:

- algunas diferencias desaparecen;
- algunas dimensiones interfieren;
- algunas combinaciones se vuelven ambiguas.

De aquí nació una de las ideas más importantes de Origami:

\[
\text{capacidad nominal}
\neq
\text{capacidad perceptualmente segura}
\]

---

# 7. TERCERA ETAPA — DE SÍMBOLOS A RELACIONES

Otra limitación apareció rápidamente.

Incrementar indefinidamente el número de símbolos no escala bien.

Era necesario representar **relaciones**.

Así apareció una estructura semejante a un:

**grafo dirigido, tipado, atribuido e incluso hipergrafo.**

Esto permitió expresar:

```text
A → B
A depende de B
A contiene B
A es consecuencia de B
A referencia B
A precede a B
A y B pertenecen a C
```

La información dejó de residir exclusivamente en los nodos.

También podía residir en:

\[
\text{relación}(A,B)
\]

o incluso:

\[
\text{relación}(A,B,C,D)
\]

Ésta fue una transición fundamental.

Origami empezó a dejar de ser un lenguaje visual para convertirse en un **lenguaje estructural**.

---

# 8. CUARTA ETAPA — HYPERFOLD

Posteriormente apareció **Origami HyperFold**, u OHF.

OHF intenta construir carriers extremadamente compactos capaces de representar una fuente compleja.

Pero OHF no debe confundirse con Origami.

La relación correcta es:

```text
ORIGAMI
│
├── semántica
├── estados
├── relaciones
├── dinámica
├── observación
├── percepción
├── máquina
│
└── OHF
    └── carrier / representación compacta
```

OHF es una línea de investigación dentro de Origami.

---

# 9. LA IDEA GENERATIVA DE HYPERFOLD

Un carrier OHF no debería convertirse en un gigantesco código QR.

Su objetivo es aprovechar estructura.

Una fuente puede descomponerse conceptualmente como:

\[
X = G(\theta) + R
\]

donde:

- \(G\) es una representación generativa;
- \(\theta\) son parámetros;
- \(R\) es un residual.

Si gran parte de la información puede producirse mediante \(G\), sólo necesitamos almacenar las desviaciones.

De aquí surgen representaciones como:

- `LITERAL`
- `REF`
- `REPEAT`
- `SLICE`
- `PATCH`
- `TRANSFORM`
- `RULE`
- `GRAPH_EXPAND`
- `MOTIF_EXPAND`
- `DEFAULT`
- `OVERRIDE`
- `RESIDUAL`
- `VERIFY`

---

# 10. REPRESENTATION TOURNAMENT

No existe una representación óptima para todas las regiones de una fuente.

Una región puede ser:

- literal;
- repetitiva;
- predecible;
- gráfica;
- jerárquica;
- transformable;
- altamente entrópica.

Origami debe poder probar distintas representaciones y escoger la mejor.

Eso dio lugar al concepto de:

**Representation Tournament.**

La función objetivo no debe considerar sólo tamaño.

Conceptualmente:

\[
Cost =
f(
size,
complexity,
perception,
dependency,
unfolding,
verification
)
\]

Una representación minúscula pero imposible de interpretar no es una buena representación.

---

# 11. SUPERINDEX

Si una representación contiene millones de elementos, no queremos recorrerla completa ante cada pregunta.

Necesitamos direccionamiento.

Por ello apareció el **SuperIndex**.

Su función conceptual es conectar:

```text
concepto
 → región
 → nodo
 → relación
 → dependencia
 → evidencia
```

Una consulta puede producir:

\[
Q
\rightarrow
Addresses(Q)
\rightarrow
DependencyClosure
\]

y solamente entonces desplegar la información necesaria.

---

# 12. SELECTIVE UNFOLDING

Ésta es una de las ideas centrales de Origami.

No queremos:

```text
carrier
→ reconstruir 100 %
→ buscar respuesta
```

Queremos:

```text
carrier
→ interpretar consulta
→ localizar direcciones
→ calcular dependencias
→ desplegar 0.01 %
→ responder
```

Dependiendo del problema, las ventanas pueden ser:

- espaciales;
- semánticas;
- temporales;
- jerárquicas;
- relacionales;
- de dependencia;
- de verificación.

Esto conecta Origami con conceptos de:

- lazy evaluation;
- sparse computation;
- graph traversal;
- retrieval;
- atención;
- locality.

---

# 13. EL GRAN CAMBIO: ORIGAMI DEJA DE SER VISUAL

La evolución más importante ocurrió cuando reconocimos que toda esta maquinaria tenía sentido incluso si eliminábamos completamente la imagen.

Entonces apareció la verdadera naturaleza de Origami.

## Origami es una máquina de estados relacional.

En su forma mínima podemos describirla como:

\[
S_{t+1}=F(S_t,C_t,R)
\]

donde:

- \(S_t\) = estado en el instante \(t\);
- \(C_t\) = contexto;
- \(R\) = reglas;
- \(F\) = función de transición.

La representación visual es otra función:

\[
P=G(S,O,\tau)
\]

donde:

- \(P\) = percepción;
- \(S\) = estado;
- \(O\) = observador;
- \(\tau\) = trayectoria de observación.

Por tanto:

\[
F \neq G
\]

La evolución del sistema y la manera de observarlo son problemas diferentes.

---

# 14. INSPIRACIÓN EN AUTÓMATAS CELULARES

Los autómatas celulares demostraron una idea extremadamente poderosa:

> reglas locales sencillas pueden generar comportamientos globales extremadamente complejos.

Origami adopta esa idea, pero no quiere limitarse a una retícula regular.

Una máquina Origami puede tener:

- nodos;
- regiones;
- celdas;
- entidades;
- capas;
- relaciones;
- grafos;
- jerarquías;
- diferentes escalas.

Las reglas pueden ser:

- locales;
- regionales;
- globales;
- temporales;
- relacionales.

---

# 15. LA EXPLOSIÓN COMBINATORIA NO ES SIEMPRE EL ENEMIGO

Normalmente los sistemas computacionales intentan eliminar la explosión combinatoria.

Origami adopta una postura diferente.

Algunas veces el enorme espacio de configuraciones **es precisamente aquello que queremos estudiar o utilizar**.

Una máquina con:

\[
n
\]

variables binarias posee:

\[
2^n
\]

configuraciones posibles.

Pero Origami puede utilizar estados mucho más ricos y relaciones entre ellos.

Esto genera enormes espacios de estados.

El objetivo no es necesariamente enumerarlos completamente.

Queremos estudiar:

- regiones del espacio;
- atractores;
- ciclos;
- estabilidad;
- metaestabilidad;
- transiciones;
- bifurcaciones;
- caos;
- configuraciones imposibles;
- estructuras emergentes.

En otras palabras:

**no destruir la complejidad, sino domesticarla.**

---

# 16. PRESENCIA Y AUSENCIA PUEDEN TENER SIGNIFICADO

Otra idea importante es que:

```text
algo presente
```

puede transportar información.

Pero también:

```text
algo ausente
```

puede transportar información.

Sin embargo Origami debe distinguir:

```text
ABSENT
UNKNOWN
LATENT
INHIBITED
CANCELLED
```

No son equivalentes.

La ausencia puede ser un estado significativo.

La incertidumbre epistemológica es otra cosa.

---

# 17. SEMÁNTICA DE ESTADOS COHERENTES

Origami introdujo posteriormente un perfil inspirado matemáticamente en conceptos de mecánica cuántica.

Esto NO significa que Origami sea una computadora cuántica.

No implica física cuántica real.

Es una **analogía matemática y operacional**.

Los estados actuales incluyen ideas como:

```text
determinate
superposed
coupled
observed
```

---

# 18. SUPERPOSICIÓN

Una entidad puede representar varias alternativas simultáneamente antes de una resolución explícita.

Conceptualmente:

\[
|\psi\rangle =
\alpha|A\rangle+
\beta|B\rangle
\]

Aquí las amplitudes pueden aportar:

- peso;
- fase;
- posibilidad de interferencia.

Pero Origami no afirma que exista una función de onda física.

Utiliza esta estructura como lenguaje de representación.

---

# 19. INTERFERENCIA

Dos contribuciones pueden combinarse constructiva o destructivamente.

Por ejemplo:

\[
1 + (-1)=0
\]

Pero:

\[
0
\]

por cancelación no significa:

```text
UNKNOWN
```

Significa:

```text
CANCELLED
```

Esta diferencia es importante.

Origami intenta preservar la **causa del estado**, no sólo su valor final.

---

# 20. COUPLED STATE

Dos entidades pueden encontrarse en un estado que no debería interpretarse correctamente si cada una se evalúa de manera independiente.

Ésta es la idea de:

```text
COUPLED
```

La relación forma parte del estado.

Por tanto:

\[
State(A,B)
\neq
State(A)+State(B)
\]

en ciertos casos.

---

# 21. OBSERVE

Origami intenta mantener una frontera explícita entre evolución y observación.

`TRANSFORM` modifica un estado.

`OBSERVE` obtiene o resuelve información de acuerdo con una política de observación.

Esto impide introducir decisiones ocultas dentro de transformaciones aparentemente neutras.

---

# 22. FOLD Y UNFOLD EN LA MÁQUINA

Estos términos dejaron de significar solamente compresión.

## UNFOLD

Puede expandir:

- alternativas;
- dependencias;
- estructuras;
- reglas;
- representaciones.

## FOLD

Puede:

- resumir;
- restringir;
- combinar;
- seleccionar;
- resolver explícitamente de acuerdo con una política.

Así, Fold/Unfold son operaciones generales sobre representación y estado.

---

# 23. ESTADO Y PERCEPCIÓN SON ORTOGONALES

Una entidad puede tener un estado perfectamente definido pero no ser perceptualmente accesible.

Por ejemplo:

```text
state = determinate
perception = latent
```

No existe contradicción.

Una dimensión describe **qué es**.

Otra describe **si puede observarse**.

---

# 24. CANALES PERCEPTUALES

Origami actualmente distingue varias familias.

## Spatial

Información disponible por organización espacial.

## Interference

Información que surge mediante interacción entre patrones.

## Depth

Información que requiere disparidad, paralaje o múltiples perspectivas.

## Temporal

Información disponible solamente a través de evolución temporal.

## Emergent

Información que pertenece a la interacción global entre componentes.

---

# 25. MOIRÉ E INTERFERENCIA

Dos patrones aparentemente simples pueden generar un patrón macroscópico nuevo.

Esto ocurre matemáticamente al combinar frecuencias espaciales próximas.

Conceptualmente:

\[
f_1-f_2
\]

puede generar una frecuencia aparente mucho menor.

Esto permite imaginar representaciones donde:

```text
capa A
```

no contiene explícitamente cierto percepto;

```text
capa B
```

tampoco;

pero:

```text
A + B
```

sí lo produce.

El significado pertenece a la relación.

---

# 26. PROFUNDIDAD Y PARALAJE

Una representación puede necesitar más de una perspectiva.

Información ausente de una proyección puede aparecer mediante:

- disparidad;
- movimiento;
- perspectiva;
- paralaje.

De aquí aparecen operaciones como:

```text
STEREO_BIND
PARALLAX_RESOLVE
```

---

# 27. TEMPORAL LATENT IMAGE

Una de las ideas más interesantes surgidas en Origami es la **Temporal Latent Image**, o TLI.

Una TLI es una estructura cuyo percepto pretendido:

- no está disponible;
- o está incompleto;

en una observación estática autorizada.

Pero puede surgir mediante una trayectoria temporal específica.

Conceptualmente:

\[
P =
G(S_{0:n}, O,\tau)
\]

No depende únicamente de una imagen.

Depende de una secuencia.

Puede involucrar:

- movimiento;
- fase;
- persistencia;
- integración temporal;
- transformación.

---

# 28. QUE ALGO SEA LATENTE NO SIGNIFICA QUE SEA INDEMOSTRABLE

Ésta fue una corrección conceptual importante.

No podemos afirmar:

> «La información existe pero nunca puede fallar en aparecer.»

Eso haría la hipótesis imposible de falsar.

Por ello Origami introdujo:

# OBSERVATION CONTRACT

Antes de ejecutar un experimento debemos declarar:

- qué buscamos;
- bajo qué condiciones;
- mediante qué observador;
- siguiendo qué trayectoria;
- durante cuánto tiempo;
- cuál sería el resultado esperado;
- qué constituye fracaso.

La observación debe tener presupuesto finito.

Después de agotarlo, la hipótesis puede fallar.

---

# 29. RESULTADOS EPISTÉMICOS

Origami no debe reducir todo a verdadero/falso.

Distinguimos estados como:

```text
PASS
FAIL
INVALID_CONTRACT
UNSUPPORTED
```

y, en percepción:

```text
KNOWN
AMBIGUOUS
UNKNOWN
INVALID
```

Estas diferencias son fundamentales.

`UNKNOWN` no significa `FALSE`.

`UNSUPPORTED` no significa `FAIL`.

`INVALID_CONTRACT` no significa que la hipótesis haya sido refutada.

---

# 30. PERCEPTION WALL

Un sistema perceptual no debería poder convertir por sí mismo una interpretación incierta en verdad exacta.

Por eso apareció la:

**Perception Wall.**

Conceptualmente:

```text
pixels
↓
perception
↓
evidence
------------------
PERCEPTION WALL
------------------
resolution
↓
execution
↓
verification
```

La capa perceptual puede decir:

```text
creo observar X
```

Pero no:

```text
X es exactamente verdadero
```

sin verificación suficiente.

---

# 31. VERIFICATION SPINE

La arquitectura moderna de Origami introduce una columna de verificación.

Una posible cadena es:

```text
Perception
↓
Resolution
↓
Execution Gate
↓
Bounded Execution
↓
Residual
↓
Verification
↓
Commit Gate
```

Sólo la evidencia apropiada puede promover una afirmación a exactitud.

Nuestro principio:

\[
FALSE\_EXACT = 0
\]

No significa que jamás existirá un bug.

Significa que el diseño debe considerar un **falso exacto** como una violación grave del sistema.

---

# 32. PERCEPTION LAB

Para desarrollar carriers visuales necesitamos medir experimentalmente qué puede leerse de forma fiable.

Por eso existe Perception Lab.

No basta con decir:

> «Se ven diferentes.»

Necesitamos controlar experimentalmente dimensiones.

Ejemplo:

```text
A = forma
B = color
```

Debemos probar:

```text
A
B
A×B
```

porque:

\[
PASS(A) \land PASS(B)
\]

no demuestra:

\[
PASS(A\times B)
\]

---

# 33. PERCEPTUAL ORTHOGONALITY

Queremos descubrir dimensiones suficientemente independientes.

La prueba conceptual consiste en modificar una dimensión manteniendo las demás constantes.

Si modificamos:

```text
forma
```

el observador debería seguir distinguiendo:

```text
color
orientación
región
relación
```

cuando corresponda.

Las interacciones negativas reducen la capacidad útil.

---

# 34. DVR Y PSS

El **Dimensional Visual Register** intenta catalogar dimensiones visuales utilizables.

El **Perceptual State Space** intenta modelar sus combinaciones.

Pero Origami distingue:

\[
PSS_{nominal}
\]

de:

\[
SAFE\_PSS
\]

`SAFE_PSS` sólo debe contener combinaciones respaldadas experimentalmente.

---

# 35. MICRO, MESO Y MACRO

No toda información aparece en la misma escala.

## MICRO

Propiedades de elementos individuales.

## MESO

Patrones regionales.

## MACRO

Estructuras globales.

Un diseño puede ser perfectamente legible a escala MICRO y perder completamente su estructura MACRO después de un resize.

Por ello deben evaluarse independientemente.

---

# 36. EL VLM COMO CANAL RUIDOSO

Una idea fundamental de Origami es tratar al observador visual como un canal imperfecto.

La cadena puede incluir:

```text
carrier
↓
resize
↓
compresión
↓
render
↓
captura
↓
modelo
```

Cada transformación puede perder información.

Esto conecta Origami directamente con teoría de comunicación.

---

# 37. CIENCIA FUNDAMENTAL DETRÁS DE ORIGAMI

Origami combina ideas procedentes de muchas disciplinas.

No debemos confundirlas con descubrimientos propios.

## Teoría de la información

Aporta:

- entropía;
- redundancia;
- información mutua;
- capacidad de canal;
- ruido;
- rate–distortion.

## Complejidad algorítmica

Aporta la idea de representar regularidades mediante descripciones cortas.

## Minimum Description Length

Ayuda a pensar una representación como:

\[
modelo + residual
\]

## Teoría de grafos

Aporta relaciones, dependencias, caminos y estructuras.

## Gramáticas formales

Aportan reglas capaces de generar estructuras complejas.

## Autómatas

Aportan estados y transiciones.

## Sistemas dinámicos

Aportan:

- atractores;
- ciclos;
- estabilidad;
- caos;
- bifurcaciones.

## Ciencia de la complejidad

Aporta emergencia y organización multiescala.

## Álgebra lineal

Aporta espacios de estados, proyecciones y transformaciones.

## Teoría de señales

Aporta frecuencia, fase, interferencia, Fourier, moiré y aliasing.

## Visión computacional

Aporta profundidad, estereoscopía, paralaje, oclusión y geometría.

## Psicofísica

Aporta discriminabilidad perceptual y umbrales.

## Teoría de detección

Aporta falsos positivos, falsos negativos, señal y ruido.

## Teoría de códigos

Aporta redundancia y corrección/detección de errores.

## Sistemas distribuidos

Aportan interacción entre procesos y comportamiento colectivo.

## Computación sparse

Aporta cálculo selectivo.

## Lazy evaluation

Aporta la idea de no materializar aquello que todavía no necesitamos.

## Modelos generativos

Aportan representación mediante predicción más residual.

## Métodos formales

Aportan invariantes, especificaciones, semántica y verificabilidad.

## Método científico

Aporta falsabilidad, hipótesis previas, protocolos y reproducibilidad.

---

# 38. QUÉ ES CIENCIA Y QUÉ ES ORIGAMI

Desde ahora todo concepto deberá etiquetarse mentalmente en una de cuatro categorías.

## A. FUNDAMENTO ESTABLECIDO

Conocimiento científico ya existente.

Ejemplo:

```text
entropía de Shannon
```

## B. ABSTRACCIÓN ADOPTADA

Concepto existente adaptado a Origami.

Ejemplo:

```text
superposición como representación computacional
```

## C. HIPÓTESIS ORIGAMI

Algo que creemos que podría funcionar pero todavía requiere evidencia.

Ejemplo:

```text
cierta combinación perceptual puede aumentar SAFE_PSS
```

## D. RESULTADO EXPERIMENTAL

Algo que Origami haya medido bajo un protocolo concreto.

Nunca deben mezclarse estas cuatro categorías.

---

# 39. REFERENCE ENGINE

Para demostrar que Origami no depende de imágenes construimos un motor de referencia.

Su propósito es ejecutar una máquina relacional determinista.

Puede:

- enumerar estados;
- aplicar reglas;
- producir trazas;
- detectar contradicciones;
- detectar puntos fijos;
- detectar ciclos;
- detectar agotamiento de presupuesto.

Esto convierte las ideas semánticas en comportamiento ejecutable.

---

# 40. EXP-001

El primer experimento formal utiliza entidades:

```text
A
B
C
D
```

con relaciones semejantes a:

```text
A requires B
B excludes C
C requires D
D couples A
```

El sistema explora configuraciones y las clasifica.

Resultados posibles:

```text
FIXED_POINT
CYCLE
CONTRADICTION
BUDGET_EXHAUSTED
```

Su objetivo no es resolver un problema práctico.

Su objetivo es probar que la máquina formal puede producir dinámicas verificables.

---

# 41. ORIGAMI COMO MÁQUINA DE EXPERIMENTACIÓN

Aquí aparece una posibilidad mucho mayor.

Si Origami puede representar:

- estado;
- reglas;
- relaciones;
- tiempo;
- observadores;

entonces puede convertirse en un laboratorio para probar sistemas abstractos.

Podemos preguntar:

```text
¿qué ocurre si modificamos esta regla?
```

y observar:

- nuevos atractores;
- ciclos;
- colapsos;
- estructuras;
- efectos emergentes.

Esto aproxima Origami a una plataforma de investigación computacional.

---

# 42. ORIGAMI Y RAZONAMIENTO

Otra dirección consiste en utilizar estados y relaciones como estructura sobre la cual realizar razonamiento.

En lugar de pedir a un modelo que mantenga todo implícitamente dentro de activaciones neuronales, parte del problema podría existir explícitamente en una máquina Origami.

Conceptualmente:

```text
modelo
↓
construye / consulta
↓
máquina Origami
↓
evolución verificable
↓
evidencia
↓
modelo
```

Origami podría convertirse así en una memoria externa estructurada y dinámica.

---

# 43. ORIGAMI Y AGENTES

Origami no es el sistema de agentes.

Eso pertenece principalmente a Tlaloc.

Pero Origami puede proporcionarles:

- estado compartido;
- relaciones;
- memoria;
- evidencia;
- estructura consultable;
- espacios de hipótesis;
- resultados experimentales.

Los agentes especializados son los **Tlaloque**.

La división correcta es:

```text
Tlaloc
→ decide / coordina / ejecuta trabajo

Origami
→ representa / evoluciona / consulta / verifica estructuras
```

---

# 44. TONAL

Tonal se encuentra un nivel por encima.

Conceptualmente:

```text
TONAL
│
├── Tlaloc
│   ├── orquestación
│   ├── comportamiento
│   ├── Tlaloque
│   └── evaluación
│
└── Origami
    ├── representación
    ├── estado
    ├── relaciones
    ├── dinámica
    ├── percepción
    └── OHF
```

Tonal es el ecosistema de composición.

Origami sigue siendo independiente.

---

# 45. EL PDF COMO CASO DE USO

Uno de los experimentos prácticos más importantes consiste en entregar un PDF completo a Origami.

El objetivo no debería ser simplemente:

```text
PDF → comprimir archivo
```

sino:

```text
PDF
↓
interpretar
↓
estructurar
↓
direccionar
↓
representar
↓
consultar
```

---

# 46. INGESTIÓN DEL PDF

Podemos construir:

```text
PDF
↓
páginas
↓
texto
↓
bloques
↓
secciones
↓
tablas
↓
imágenes
↓
metadatos
↓
relaciones
```

Posteriormente generar:

- índices de página;
- índices de términos;
- índices semánticos;
- referencias;
- grafos;
- estructuras Origami.

---

# 47. CONSULTA DEL PDF

Una pregunta podría seguir:

```text
QUESTION
↓
QUERY INTERPRETATION
↓
SUPERINDEX
↓
LOGICAL ADDRESSES
↓
DEPENDENCY CLOSURE
↓
SELECTIVE UNFOLD
↓
EVIDENCE
↓
ANSWER
```

Así Origami no necesita reconstruir el libro completo ante cada pregunta.

---

# 48. EXACT CONTENT Y EXACT SOURCE

Debemos distinguir dos objetivos.

## EXACT_CONTENT

Recuperar correctamente la información semántica o textual.

## EXACT_SOURCE

Recuperar exactamente los bytes originales.

Son problemas diferentes.

Una aplicación de consulta documental puede necesitar `EXACT_CONTENT` sin necesidad de reconstrucción binaria del PDF original.

OHF puede investigar ambos perfiles sin mezclarlos.

---

# 49. OBJETIVO DE OHF

OHF intenta estudiar hasta qué punto una fuente altamente estructurada puede convertirse en un carrier extremadamente compacto.

Existe un objetivo experimental muy agresivo alrededor de:

```text
≤ 500 KB
```

Pero éste debe entenderse correctamente.

No afirmamos:

```text
cualquier 5 GB
→ 500 KB
```

Eso contradice límites fundamentales de teoría de la información para datos arbitrarios.

La verdadera pregunta es:

> ¿Cuánta estructura generativa, repetición, relación y conocimiento compartido podemos explotar antes de que el residual domine?

Eso sí es una pregunta científica legítima.

---

# 50. ARQUITECTURA CONCEPTUAL DE OHF

El lado emisor puede pensarse como:

```text
SOURCE
↓
INGESTION
↓
CANONICAL SOURCE MODEL
↓
STRUCTURE ANALYSIS
↓
SEMANTIC ANALYSIS
↓
ENTROPY ANALYSIS
↓
SOURCE GRAPH
↓
PATTERN DISCOVERY
↓
GRAMMAR
↓
TRANSFORMS
↓
MOTIFS
↓
FOLD CANDIDATES
↓
REPRESENTATION TOURNAMENT
↓
GENERATIVE IR
↓
RESIDUAL
↓
SUPERINDEX
↓
ATTENTION PLAN
↓
VISUAL COMPILER
↓
ROSETTA
↓
CARRIER
```

---

# 51. RECEPCIÓN OHF

El lado receptor:

```text
CARRIER
↓
BOOT
↓
ROSETTA
↓
SUPERINDEX
↓
QUERY PLAN
↓
ATTENTION ROUTER
↓
PERCEPTION
↓
EVIDENCE
↓
PERCEPTION WALL
↓
RESOLUTION
↓
EXECUTION
↓
RESIDUAL
↓
VERIFICATION
↓
COMMIT
↓
ANSWER
```

---

# 52. MODOS DE OPERACIÓN

Origami/OHF distingue tres perfiles importantes.

## Native

El modelo recibe esencialmente:

- carrier;
- instrucciones;
- pregunta.

No dispone de decodificador privilegiado.

Sirve para medir qué puede interpretar realmente un modelo multimodal.

## Computational

Puede utilizar:

- código;
- hashes;
- píxeles;
- estructuras exactas;
- decodificadores.

Sirve como referencia determinista.

## Hybrid

Combina:

```text
percepción visual
+
ejecución determinista
+
verificación
```

Probablemente sea uno de los perfiles más prácticos.

---

# 53. ORIGAMI NO DEBE CONVERTIRSE EN UNA VM GENERAL

Aunque Origami posee operaciones ejecutables, no queremos convertirlo por defecto en una máquina virtual arbitraria.

La ejecución debe ser:

- declarativa;
- acotada;
- verificable;
- determinista cuando corresponda;
- limitada por presupuesto.

Esto reduce:

- comportamiento impredecible;
- complejidad innecesaria;
- superficie de errores.

---

# 54. MULTIESCALA

Origami debe poder trabajar simultáneamente en diferentes escalas.

Por ejemplo:

```text
local
regional
global
```

o:

```text
fast
medium
slow
```

Esto abre una dirección especialmente importante.

Un proceso local rápido puede resolver detalles mientras otro proceso más lento mantiene coherencia global.

---

# 55. TIEMPO COMO DIMENSIÓN FUNDAMENTAL

El tiempo no debe considerarse solamente:

```text
frame 1
frame 2
frame 3
```

Una estructura puede poseer propiedades que sólo existen debido a su evolución.

Por tanto:

\[
Property(S_t)
\]

puede ser insuficiente.

Puede necesitarse:

\[
Property(S_{t_0:t_n})
\]

La propiedad pertenece a la trayectoria.

---

# 56. EMERGENCIA

Otra idea central es que algunas propiedades no pertenecen a ninguna parte individual.

Podemos tener:

\[
P(A)=0
\]

\[
P(B)=0
\]

pero:

\[
P(A,B)=1
\]

La propiedad existe en la interacción.

Este principio aparece repetidamente:

- moiré;
- sistemas dinámicos;
- grafos;
- comportamiento colectivo;
- Gestalt;
- swarm systems;
- estructuras temporales.

---

# 57. ORIGAMI COMO LENGUAJE DE RELACIONES

A largo plazo Origami podría describir un sistema mediante:

```text
ENTITIES
RELATIONS
STATES
RULES
OBSERVERS
TRAJECTORIES
INVARIANTS
EVIDENCE
```

Esto constituye una forma extremadamente general de representar problemas.

---

# 58. OBJETIVO PROFUNDO

El objetivo profundo de Origami no consiste en producir imágenes bonitas.

Ni siquiera consiste únicamente en comprimir información.

Queremos investigar si podemos construir un sistema donde **representación, computación y percepción formen distintas vistas del mismo estado estructurado**.

Idealmente:

\[
Representation
\leftrightarrow
Computation
\leftrightarrow
Dynamics
\leftrightarrow
Perception
\]

sin confundirlas.

---

# 59. ORIGAMI COMO MEMORIA COMPUTACIONAL

Una memoria normalmente almacena valores.

Origami busca almacenar además:

- relaciones;
- contexto;
- reglas;
- dependencias;
- historia;
- evidencia;
- procedimientos de reconstrucción.

Podría pensarse como:

**memoria que sabe cómo desplegarse.**

---

# 60. ORIGAMI COMO ESPACIO DE TRABAJO PARA IA

A largo plazo un modelo podría utilizar Origami como un espacio externo donde:

1. representar hipótesis;
2. conectar evidencia;
3. ejecutar transformaciones;
4. comparar alternativas;
5. detectar contradicciones;
6. simular consecuencias;
7. conservar trazabilidad.

Esto podría reducir parte de la dependencia del razonamiento puramente implícito de un LLM.

---

# 61. ORIGAMI COMO LENGUAJE ENTRE MÁQUINAS

Otra posibilidad consiste en que Origami sea utilizado como contrato estructurado entre sistemas.

Por ejemplo:

```text
LLM
↓
Origami
↓
agente
↓
Origami
↓
verificador
```

En vez de intercambiar enormes cantidades de lenguaje natural ambiguo, podrían intercambiar estados estructurados y evidencias.

---

# 62. ORIGAMI COMO LABORATORIO DE NUEVAS ARQUITECTURAS

Origami también puede utilizarse para experimentar con ideas que después puedan incorporarse a modelos.

Ejemplos:

- múltiples escalas temporales;
- memoria jerárquica;
- estados acoplados;
- dinámica relacional;
- atención direccionable;
- representación generativa;
- cómputo sparse;
- estructuras emergentes.

Origami puede actuar como un entorno donde estas ideas se prueben antes de intentar introducirlas dentro de una red neuronal.

---

# 63. OBJETIVOS A CORTO PLAZO

La etapa actual debe consolidar los fundamentos.

Prioridades:

### Formalizar la máquina

Integrar completamente:

- estados;
- relaciones;
- reglas;
- observación;
- interferencia;
- temporalidad.

### Construir fixtures deterministas

Especialmente para:

- interferencia;
- profundidad;
- TLI.

### Expandir Reference Engine

Hasta que pueda ejecutar una parte creciente de la semántica formal.

### Mantener `UNSUPPORTED`

Si una operación todavía no existe, debe declararse explícitamente.

Nunca fingirse.

---

# 64. OBJETIVOS A MEDIANO PLAZO

Construir una verdadera plataforma Origami capaz de:

```text
SOURCE
↓
ORIGAMI REPRESENTATION
↓
MACHINE
↓
INDEX
↓
QUERY
↓
SELECTIVE EXECUTION
↓
EVIDENCE
```

Y validar:

- documentación;
- grafos;
- sistemas dinámicos;
- datos estructurados;
- problemas sintéticos.

---

# 65. OBJETIVOS DE OHF

En paralelo:

- mejorar DVR;
- mejorar PSS;
- construir SAFE_PSS;
- explorar Glyph Calculus;
- explorar SAFE_MICRO_ISA;
- Context SIMD;
- Macro-Gestalt;
- mejorar representation tournament;
- mejorar SuperIndex;
- probar modelos específicos;
- fortalecer Native;
- fortalecer Hybrid.

---

# 66. OBJETIVO DOCUMENTAL

Queremos poder entregar:

```text
libro
paper
manual
documentación
repositorio
```

y construir una representación Origami consultable.

El usuario debería poder preguntar:

```text
¿qué dice acerca de X?
¿en qué página?
¿qué conceptos se relacionan?
¿qué contradicciones existen?
¿qué depende de qué?
```

sin tener que recorrer manualmente toda la fuente.

---

# 67. OBJETIVO DE RAZONAMIENTO

Más adelante queremos comprobar si una máquina Origami puede ayudar a realizar tareas que requieren:

- razonamiento multietapa;
- planificación;
- dependencia causal;
- búsqueda de hipótesis;
- simulación;
- comparación de escenarios.

---

# 68. OBJETIVO DE SWARM

Tlaloc podrá coordinar múltiples Tlaloque que utilicen Origami como espacio común.

Por ejemplo:

```text
Tlaloque A
→ propone hipótesis

Tlaloque B
→ busca evidencia

Tlaloque C
→ intenta refutarla

Tlaloque D
→ simula consecuencias

Origami
→ mantiene estado/evidencia/relaciones

Tlaloc
→ coordina
```

Esto puede permitir sistemas multiagente más disciplinados que simples conversaciones entre agentes.

---

# 69. OBJETIVO A LARGO PLAZO

La ambición mayor puede resumirse así:

> construir una representación computacional donde grandes espacios de información y estado puedan plegarse en estructuras compactas, direccionables y generativas; evolucionar mediante reglas; revelar propiedades espaciales, temporales y emergentes; desplegar únicamente las partes relevantes para una consulta; y producir resultados acompañados de evidencia verificable.

---

# 70. LO QUE ORIGAMI NO DEBE PROMETER

Origami no debe afirmar sin evidencia:

- compresión arbitraria ilimitada;
- computación cuántica física;
- conciencia;
- razonamiento perfecto;
- percepción perfecta;
- independencia perceptual sin pruebas;
- propiedades emergentes no falsables;
- exactitud derivada solamente de un VLM;
- capacidad matemática equivalente a capacidad observable.

---

# 71. PRINCIPIO DE HONESTIDAD EXPERIMENTAL

Ante una nueva idea preguntaremos siempre:

### ¿Qué representa?

### ¿Cuál es el estado?

### ¿Cuáles son las relaciones?

### ¿Cuáles son las reglas?

### ¿Cómo evoluciona?

### ¿Qué puede observarse directamente?

### ¿Qué emerge?

### ¿Qué observador necesita?

### ¿Qué trayectoria necesita?

### ¿Puede fallar?

### ¿Cómo sabremos que falló?

### ¿Puede direccionarse?

### ¿Puede desplegarse selectivamente?

### ¿Cómo se verifica?

### ¿Qué evidencia tenemos?

Sólo después preguntaremos:

> ¿Cómo debería visualizarse?

---

# 72. PRINCIPIO DE SEPARACIÓN

Debemos mantener siempre estas capas separadas:

```text
SEMANTICS
↓
STATE
↓
DYNAMICS
↓
OBSERVATION
↓
PERCEPTION
↓
REPRESENTATION
↓
CARRIER
```

Una implementación puede conectarlas.

Pero ninguna debe confundirse con las demás.

---

# 73. PRINCIPIO DE EVIDENCIA

Una característica puede atravesar aproximadamente estas etapas:

```text
IDEA
↓
HYPOTHESIS
↓
SPEC
↓
DETERMINISTIC TEST
↓
CONTROLLED EXPERIMENT
↓
PERCEPTION TEST
↓
REGRESSION
↓
PROMOTED
```

No debería saltarse directamente:

```text
IDEA → FEATURE
```

---

# 74. PRINCIPIO DE REPRODUCIBILIDAD

Cada experimento importante debe poder registrar:

- especificación;
- versión;
- configuración;
- seed;
- hashes;
- resultados;
- métricas;
- errores;
- evidencia;
- decisión.

Esto permite saber exactamente qué produjo un resultado.

---

# 75. PRINCIPIO DE CONTROL DE CAMBIOS

Cuando Origami cambie debemos registrar:

```text
qué cambió
estado anterior
estado posterior
evidencia
tests
regresiones
impacto
decisión
```

No queremos que una idea experimental cambie silenciosamente la semántica del sistema.

---

# 76. ORIGAMI ACTUAL

Actualmente Origami debe entenderse aproximadamente como:

```text
ORIGAMI
│
├── Formal Core
│
│   ├── state
│
│   ├── relations
│   ├── rules
│   ├── dynamics
│   └── observation contracts
│
├── State Semantics
│   ├── determinate
│   ├── superposed
│   ├── coupled
│   ├── observed
│   ├── Fold
│   ├── Unfold
│   ├── Transform
│   ├── Observe
│   └── Interfere
│
├── Perceptual Semantics
│   ├── spatial
│   ├── interference
│   ├── depth
│   ├── temporal
│   └── emergent
│
├── Reference Machine
│
├── Experimental Framework
│
├── Query / Addressing
│   ├── SuperIndex
│   ├── dependency closure
│   └── selective unfolding
│
└── OHF
    ├── generative IR
    ├── residual
    ├── DVR
    ├── PSS
    ├── Perception Lab
    ├── visual compiler
    ├── Rosetta
    └── carrier
```

---

# 77. LA FRASE MÁS CORTA POSIBLE

Si hubiera que explicarle Origami a alguien en una sola frase:

> **Origami es un lenguaje y una máquina para plegar información, relaciones y dinámica en estructuras direccionables que pueden evolucionar y desplegar sólo aquello necesario para observar, consultar o verificar algo.**

---

# 78. LA FRASE CIENTÍFICA

En términos más técnicos:

> **Origami investiga una representación computacional relacional, generativa, dinámica, multiescala y parcialmente observable, con semántica explícita de estados, operaciones acotadas, direccionamiento selectivo, canales perceptuales y verificación independiente.**

---

# 79. LA VISIÓN FINAL

La visión no es fabricar un nuevo formato de imagen.

No es construir un nuevo algoritmo de compresión.

No es construir un nuevo Transformer.

No es construir simplemente otra base vectorial.

No es construir un sistema de agentes.

Origami intenta explorar algo más fundamental:

> **una forma diferente de representar computacionalmente información y estado, en la que estructura, relaciones, reglas, tiempo, percepción y reconstrucción sean elementos de primera clase.**

Si funciona, una misma representación podría ser:

```text
memoria
+
estado
+
grafo
+
programa restringido
+
espacio dinámico
+
índice
+
carrier
+
fuente de evidencia
```

Y ése es el territorio científico y de ingeniería que Origami pretende explorar.

---

# 80. REGLA PARA EL FUTURO DEL PROYECTO

Toda nueva propuesta deberá ubicarse dentro de este compendio.

No volveremos a redefinir Origami cada vez que aparezca una idea.

Una nueva idea deberá responder:

```text
¿Es parte del Core?

¿Es una nueva semántica?

¿Es una operación?

¿Es una forma de representación?

¿Es percepción?

¿Es un experimento?

¿Es una optimización?

¿Es parte de OHF?

¿Pertenece realmente a Tlaloc?

¿Pertenece realmente a Tonal?

¿Es fundamento científico?

¿Es analogía?

¿Es hipótesis?

¿Está demostrada?
```

De esta manera Origami podrá crecer sin convertirse en una colección de ideas pegadas entre sí.

La arquitectura deberá evolucionar.

La investigación podrá cambiar de dirección.

Algunas hipótesis serán descartadas.

Nuevos experimentos podrán obligarnos a modificar conceptos.

Pero el proyecto conservará una historia, una taxonomía y una identidad coherentes.

---

# DEFINICIÓN CANÓNICA ACTUAL

**Origami es un sistema experimental de representación y computación estructural cuyo objetivo es describir estados, relaciones y dinámicas mediante estructuras generativas y direccionables; permitir propiedades directas, latentes, temporales y emergentes; ejecutar transformaciones acotadas sobre dichas estructuras; desplegar selectivamente sólo la información necesaria para una consulta u observación; proyectar opcionalmente esos estados sobre canales perceptuales —incluyendo representaciones visuales mediante OHF— y separar rigurosamente percepción, inferencia, ejecución, evidencia y verificación.**

Su objetivo último es investigar si esa combinación puede convertirse en una nueva clase de memoria y espacio computacional para máquinas, modelos de IA, agentes y sistemas de razonamiento.