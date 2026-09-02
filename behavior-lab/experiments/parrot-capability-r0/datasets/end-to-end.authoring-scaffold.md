# P0 authoring scaffold — Understanding Large Language Models Towards Rigorous and Targeted Interpretability Using Probing Classifiers.pdf

Hand-write the remaining questions into `end-to-end.jsonl` (start from
`end-to-end.draft.jsonl`, which holds the auto-generated ones). Each question
must be answerable **only** from the evidence quoted below. Schema:
`datasets/SCHEMA.md`. Then: `validate --stage end_to_end` → `freeze --scope stage`.

Target: 30 questions over these 10 pages — 6 each of locate / entity / factual / numeric / synthesis.

## Page 28 — ""

- address: `ohf://fold-bench/pages/000028`  ·  page cid: `0a0ffb4e6bcc6a063f2b0ef3b9d8c7ea5c13261eb8f958eae83855fa66b5dd2d`
- rendered page: `end-to-end/scaffold-images/p28.png`
- numbers in the prose (skip code):
    - `1` — For a sequence of N tokens ( t 1 , t 2 , .
    - `1,024` — ELMO has a character-based word representation layer with 512 dimensions and 2 bi-LSTM hidden layers with 1,024 units.
    - `12` — The standard BERT base model consists of 12 layers, while the BERT large model has 24 layers.
    - `16` — This is done multiple times in parallel (12 times in the case of BERT base ; 16 times for BERT large ) to be able to capture richer features from the representa…
    - `2` — For a sequence of N tokens ( t 1 , t 2 , .
    - `2.3` — 2.3.2 BERT BERT (Devlin et al.
    - `2017` — 2017) that contextualizes the word representations with multi-head self-attention and fully connected layers.
    - `2019` — 2019) is based on a Transformer model’s encoder (Vaswani et al.
    - `24` — The standard BERT base model consists of 12 layers, while the BERT large model has 24 layers.
    - `3` — The token representation commonly used in downstream tasks is either the top layer or a task-specific weighted sum of the 3 internal layers of the LSTM.
    - `512` — ELMO has a character-based word representation layer with 512 dimensions and 2 bi-LSTM hidden layers with 1,024 units.
- candidate noun phrases: "BERT BERT", "EARNING ELMO"
- auto-generated for this page:
    - [numeric] Fill the blank using only the page. Answer with the number only. "ELMO has a character-based word representation layer w… → "512"
    - [numeric] Fill the blank using only the page. Answer with the number only. "The standard BERT base model consists of _____ layers,… → "12"
    - [synthesis] The page states both "12 layers" and "24 layers". Which is the larger quantity? Answer with that quantity exactly as wri… → "24 layers"
    - [entity] Which term from the page is described as: "provides the representation with context, as it aggregates information from t… → "self-attention module"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
2. L
ANGUAGE
R
EPRESENTATION
L
EARNING
ELMO is based on a bidirectional LSTM (Hochreiter and Schmidhuber 1997) consisting
of a forward and a backward language model that predicts the next (respectively the
previous) token conditioned on the LSTM accumulation of the preceding (respectively
the future) tokens, maximizing the log-likelihood of both directions. For a sequence of
N
tokens
(
t
1
,
t
2
, . . . ,
t
N
)
and the parameters
Θ
x
of the token representation,
Ý
Ñ
Θ
LSTM
of
the forward-pass LSTM predicting from the left-side context,
Ð
Ý
Θ
LSTM
of the backward-
pass LSTM predicting from the right-side context, and
Θ
s
of the softmax layer, ELMo
maximises:
N
¸
k
=
1
(
log
p
(
t
k
|
t
1
, . . . ,
t
k
1
;
Θ
x
,
Ý
Ñ
Θ
LSTM
,
Θ
s
)
+
log
p
(
t
k
|
t
k
+
1
, . . . ,
t
N
;
Θ
x
,
Ð
Ý
Θ
LSTM
,
Θ
s
) )
.
ELMO has a character-based word representation layer with 512 dimensions and 2
bi-LSTM hidden layers with 1,024 units. The token representation commonly used
in downstream tasks is either the top layer or a task-specific weighted sum of the 3
internal layers of the LSTM. In the latter case, the weights for each layer are learned by
the downstream model, while the parameters of the layers themselves remain frozen.
ELMo is trained on the One Billion Word Benchmark, a sentence-level English-
language dataset in the news domain with, as the name says, approximately one
billion words (Chelba et al. 2013).
2.3.2
BERT
BERT (Devlin et al. 2019) is based on a Transformer model’s encoder (Vaswani et al.
2017) that contextualizes the word representations with multi-head self-attention and
fully connected layers. Its architecture became the basis of many more Transformer-
based language representations. Transformers proved to be more successful than
other architectures like RNNs because they scale up to very deep models and the self-
attention makes them more successful at catching long-range interactions of tokens
(Bommasani et al. 2021).
After the input to the BERT model is tokenised, the embedding of the token itself
is enriched with an encoding of its position in the input span, as the Transformer
architecture does not natively model word order, and an encoding indicating which
sentence the token belongs to, as BERT can process up to two sentences at a time.
The standard BERT
base
model consists of 12 layers, while the BERT
large
model has
24 layers. The core components of each layer are a multi-head self-attention module
and a fully connected layer. The self-attention module provides the representation
with context, as it aggregates information fro
…
```
</details>

## Page 30 — ""

- address: `ohf://fold-bench/pages/000030`  ·  page cid: `b1dd2a738b474c7d1a0eaaac3986d85928f6055bf95688a13402ace5791b7f64`
- rendered page: `end-to-end/scaffold-images/p30.png`
- numbers in the prose (skip code):
    - `1.5` — 2020) adopts GPT-2’s architecture and objective, but it represents an enormous upscaling, growing from (at most) 1.5 billion to 175 billion parameters for the…
    - `12` — There are GPT-2 models in four different sizes, with the smallest one, like BERT, consisting of 12 layers and 12 self-attention heads, and the largest of 48 lay…
    - `128` — The 175 billion parameters are spread over 96 layers, and the model has 128 attention heads.
    - `175` — 2020) adopts GPT-2’s architecture and objective, but it represents an enormous upscaling, growing from (at most) 1.5 billion to 175 billion parameters for the…
    - `2` — L ANGUAGE R EPRESENTATION L EARNING Figure 2.3: Comparison of the high-level architectures of BERT and GPT-3: The constrained multi-head self-attention of the T…
    - `2.3` — L ANGUAGE R EPRESENTATION L EARNING Figure 2.3: Comparison of the high-level architectures of BERT and GPT-3: The constrained multi-head self-attention of the T…
    - `2019` — 2019), was the last GPT model with a full public release of the parameters.
    - `2020` — 2020) adopts GPT-2’s architecture and objective, but it represents an enormous upscaling, growing from (at most) 1.5 billion to 175 billion parameters for the…
    - `2021` — 2021), which has a measurable effect on the performance of models (Magar and Schwartz 2022).
    - `2022` — 2021), which has a measurable effect on the performance of models (Magar and Schwartz 2022).
    - `25` — There are GPT-2 models in four different sizes, with the smallest one, like BERT, consisting of 12 layers and 12 self-attention heads, and the largest of 48 lay…
    - `3` — L ANGUAGE R EPRESENTATION L EARNING Figure 2.3: Comparison of the high-level architectures of BERT and GPT-3: The constrained multi-head self-attention of the T…
    - `4` — 2.3.4 GPT-3 GPT-3 (Brown et al.
    - `40` — (2019) state that it contains approximately 8 million documents, and 40 gigabytes of text.
    - `48` — There are GPT-2 models in four different sizes, with the smallest one, like BERT, consisting of 12 layers and 12 self-attention heads, and the largest of 48 lay…
    - `8` — (2019) state that it contains approximately 8 million documents, and 40 gigabytes of text.
    - `9` — (2019) which has 9 billion tokens.
    - `96` — The 175 billion parameters are spread over 96 layers, and the model has 128 attention heads.
- candidate noun phrases: "EARNING Figure", "GPT-3 GPT-3"
- auto-generated for this page:
    - [numeric] Fill the blank using only the page. Answer with the number only. "There are GPT-_____ models in four different sizes, wi… → "2"
    - [numeric] Fill the blank using only the page. Answer with the number only. "2020) adopts GPT-2’s architecture and objective, but… → "1.5"
    - [numeric] Fill the blank using only the page. Answer with the number only. "The 175 billion parameters are spread over 96 layers, … → "128"
    - [synthesis] The page states both "2 models" and "3 models". Which is the larger quantity? Answer with that quantity exactly as writt… → "3 models"
    - [synthesis] The page states both "12 layers" and "48 layers". Which is the larger quantity? Answer with that quantity exactly as wri… → "48 layers"
    - [synthesis] The page states both "25 attention heads" and "128 attention heads". Which is the larger quantity? Answer with that quan… → "128 attention heads"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
2. L
ANGUAGE
R
EPRESENTATION
L
EARNING
Figure 2.3:
Comparison of the high-level architectures of BERT and GPT-3: The
constrained multi-head self-attention of the Transformer decoder caps the connections
to the preceding tokens in GPT-2. Figure by Devlin et al. (2019).
GPT-2’s architecture and objective were introduced for the original GPT model by
Radford et al. (2018). Its successor, GPT-2 (Radford et al. 2019), was the last GPT model
with a full public release of the parameters. There are GPT-2 models in four different
sizes, with the smallest one, like BERT, consisting of 12 layers and 12 self-attention
heads, and the largest of 48 layers and 25 attention heads.
GPT-2 is trained on the WebText corpus that is also introduced in Radford et al. (2019).
WebText is a scrape of outbound links from the internet forum Reddit. Radford et al.
(2019) state that it contains approximately 8 million documents, and 40 gigabytes of
text. While the original dataset is not shared and the exact number of training tokens
therefore unknown, there exists an open replication by Gokaslan et al. (2019) which
has 9 billion tokens.
2.3.4
GPT-3
GPT-3 (Brown et al. 2020) adopts GPT-2’s architecture and objective, but it represents
an enormous upscaling, growing from (at most) 1.5 billion to 175 billion parameters
for the largest of the 8 GPT-3 models. The 175 billion parameters are spread over 96
layers, and the model has 128 attention heads.
The most groundbreaking property of GPT-3 is the ability to do
few-shot learning
, also
called
in-context learning
: To adapt the model to a new task, no parameter updates are
necessary; providing a limited number of task demonstration examples in the input is
sufficient. Some success is even reported for
one-shot
and
zero-shot
transfer where only
one or no demonstration example is provided to the model when solving a new task.
It appears that, with sufficient scale, autoregressive pre-training is sufficient to infer
the structure of many tasks. It also improved the performance over smaller models.
However, it is well-documented that the training corpus of GPT-3 has a significant
amount of contamination with common benchmark tasks (Brown et al. 2020; Dodge
et al. 2021), which has a measurable effect on the performance of models (Magar and
Schwartz 2022). Therefore, comparisons have to be done with caution.
```
</details>

## Page 31 — ""

- address: `ohf://fold-bench/pages/000031`  ·  page cid: `75b2482175c16552cdf05ba3ede0d20acf557520fedccc579505cc179c127b48`
- rendered page: `end-to-end/scaffold-images/p31.png`
- numbers in the prose (skip code):
    - `1` — 2020), WebText2 (an expanded version of the GPT-2 training set), two internet-based corpora named Books1 and Books2 (with no further details released), and Engl…
    - `2` — 2020), WebText2 (an expanded version of the GPT-2 training set), two internet-based corpora named Books1 and Books2 (with no further details released), and Engl…
    - `2.3` — 2.3.5 GPT-3.5 and GPT-4 GPT-3.5 and GPT-4 (OpenAI 2023) are, as the names imply, the successors of GPT-3, and the base of the now-famous ChatGPT model.
    - `2.4` — Contextualized Language Representations Figure 2.4: The three steps of aligning an LLM: Instruction fine-tuning, training the reward model, and reinforcement le…
    - `2020` — 2020), WebText2 (an expanded version of the GPT-2 training set), two internet-based corpora named Books1 and Books2 (with no further details released), and Engl…
    - `2021` — 2021) and reinforcement learning from human feedback (RLHF; Ouyang et al.
    - `2022` — 2022) is given in Figure 2.4.
    - `2023` — 2.3.5 GPT-3.5 and GPT-4 GPT-3.5 and GPT-4 (OpenAI 2023) are, as the names imply, the successors of GPT-3, and the base of the now-famous ChatGPT model.
    - `3` — The training corpus for GPT-3 has 300B tokens and consists of is a filtered version of the web archive corpus CommonCrawl (Raffel et al.
    - `3,` — 2.3.5 GPT-3.5 and GPT-4 GPT-3.5 and GPT-4 (OpenAI 2023) are, as the names imply, the successors of GPT-3, and the base of the now-famous ChatGPT model.
    - `3.5` — 2.3.5 GPT-3.5 and GPT-4 GPT-3.5 and GPT-4 (OpenAI 2023) are, as the names imply, the successors of GPT-3, and the base of the now-famous ChatGPT model.
    - `300` — The training corpus for GPT-3 has 300B tokens and consists of is a filtered version of the web archive corpus CommonCrawl (Raffel et al.
    - `4` — 2.3.5 GPT-3.5 and GPT-4 GPT-3.5 and GPT-4 (OpenAI 2023) are, as the names imply, the successors of GPT-3, and the base of the now-famous ChatGPT model.
    - `5` — 2.3.5 GPT-3.5 and GPT-4 GPT-3.5 and GPT-4 (OpenAI 2023) are, as the names imply, the successors of GPT-3, and the base of the now-famous ChatGPT model.
- candidate noun phrases: "English-language Wikipedia", "GPT-4 GPT-3"
- auto-generated for this page:
    - [factual] Fill the blank using only the page. Answer with the missing words only. "2020), WebText2 (an expanded version of the GPT… → "English-language Wikipedia"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
2.3. Contextualized Language Representations
Figure 2.4:
The three steps of aligning an LLM: Instruction fine-tuning, training the
reward model, and reinforcement learning. Figure taken from Ouyang et al. (2022).
The training corpus for GPT-3 has 300B tokens and consists of is a filtered version of
the web archive corpus CommonCrawl (Raffel et al. 2020), WebText2 (an expanded
version of the GPT-2 training set), two internet-based corpora named Books1 and
Books2 (with no further details released), and English-language Wikipedia.
2.3.5
GPT-3.5 and GPT-4
GPT-3.5 and GPT-4 (OpenAI 2023) are, as the names imply, the successors of GPT-3,
and the base of the now-famous
ChatGPT
model. They have greatly improved zero-
shot capabilities compared to GPT-3, probably due to
instruction fine-tuning
(IFT; Wei
et al. 2021) and
reinforcement learning from human feedback
(RLHF; Ouyang et al. 2022).
Those techniques can teach a model to better follow instructions and align them with
the users’ expectations. After the general language modelling pre-training phase, the
model is fine-tuned on a supervised dataset containing a diverse set of instructions
and demonstrations of desired model behaviours. With a sufficiently diverse set of
tasks in the instructions, the model’s performance is increased even on tasks unseen
during IFT (Wei et al. 2021). In the RLHF step, human ratings of model outputs are
used as a reward signal to the model. A reward model is trained on human rankings
of multiple output candidates, which is then used to train the model via reinforcement
learning. An overview of the process as introduced in the OpenAI’s InstructGPT paper
(Ouyang et al. 2022) is given in Figure 2.4. The whole process is also called
alignment
,
as it does not only improve the zero-shot performance of the model but also aligns its
outputs better with human expectations.
However, which techniques are used for GPT-3.5 and GPT-4 is speculative. While key
information about the GPT-3 model, like the type and amount of the training data
```
</details>

## Page 37 — "3.2 Structural Probes"

- address: `ohf://fold-bench/pages/000037`  ·  page cid: `6263ec09fd6d792c8810ab1336d23c879ac676abcbb5dc23f4cbc5afaa1e6912`
- rendered page: `end-to-end/scaffold-images/p37.png`
- numbers in the prose (skip code):
    - `2` — on their word2vec model (Mikolov et al.
    - `3.2` — 3.2 Structural Probes Another line of probing assumes the geometric properties of word representation vectors to be meaningful as they reflect similarities of t…
- candidate noun phrases: "Semantic-Syntactic Word Relationship", "Structural Probes", "Structural Probes Another"
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "3.3 Mechanistic Interpretation" or "3.2 Structur… → "3.2 Structural Probes"
    - [factual] Fill the blank using only the page. Answer with the missing words only. "The authors create a data set of various kinds … → "Semantic-Syntactic Word Relationship"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
3.2. Structural Probes
A cat is [MASK] than a mouse.
We expect the probability of the token
larger
to be higher than the probability of
smaller
:
P
(
larger
)
¡
P
(
smaller
)
.
The advantage of zero-shot probes is their simplicity: No training is needed, we can
test the model directly without any modifications. Besides being easy to use, the fact
that they are a direct probe of the model, without additional learning parameters
that could influence the outcome, reduces the risk of the probe being influenced by
external factors: The ability that we test for cannot be learned at probing time. The
only model-external factor that influences the results is the choice of probing data,
which cannot be avoided in any known setup.
A downside is that zero-shot probing is limited to certain tasks that can fit into the
language modelling objective and where a comparison of output probabilities can be
interpreted in a meaningful way. And even though no learning is happening in zero-
shot probes, the design of the prompts affects the outcome. Success in such probes
is context-dependent; the models often fail where discrepancy from their training
distribution becomes too large (Talmor et al. 2020). This can make it easy to draw
conclusions (both positive and negative) that do not generalize to other experimental
designs.
3.2
Structural Probes
Another line of probing assumes the geometric properties of word representation
vectors to be meaningful as they reflect similarities of the words’ usage in the training
data. It is the first of three interpretability methods that attempt to understand the
vector space formed by model activations.
A classical example of structural probes are word analogy tasks such as the widely
known work by Mikolov et al. on their word2vec model (Mikolov et al. 2013a,b).
They show that simple addition and subtraction of word vectors can (sometimes) give
meaningful results, such as in their famous
king:queen
example: When subtracting
the vector for
man
from the word for
king
, and adding the vector for
woman
, the closest
word vector to the resulting vector is the one for
queen
.
vector(”king”) - vector(”man”) + vector(”woman”)
≈
vector(”queen”)
.
The authors create a data set of various kinds of semantic and syntactic analogies
called
Semantic-Syntactic Word Relationship
. It includes quadruples from domains such
as country and capital, adjective and adverb, and singular and plural. They assume
that a good word representation model should perform well on this data set. This
assumption has been disputed, most influentially by
…
```
</details>

## Page 38 — "3.3 Mechanistic Interpretation"

- address: `ohf://fold-bench/pages/000038`  ·  page cid: `f001cb7ebc51c9b887df5994bfb6a50b68c51009e92ac24522478b092e500575`
- rendered page: `end-to-end/scaffold-images/p38.png`
- numbers in the prose (skip code):
    - `2` — Ethayarajh (2019) investigates embeddings of BERT, ELMo, and GPT-2 with respect to how contextual they are.
    - `2014` — 2014; Levy and Goldberg 2014) show that the relational similarities are not exclusive to neural word embeddings but can also occur when using count-based word e…
    - `2017` — (2017) also show experimentally that the word analogies only work when the source and target vectors are close to each other.
    - `2019` — Example by Hewitt and Manning (2019).
    - `3.1` — I NTERPRETATION Figure 3.1: Syntax tree recovered from BERT representations with a structural probe.
    - `3.3` — 3.3 Mechanistic Interpretation Closely related to the structural probes, mechanistic interpretability aims at a fine- grained understanding of models at the neu…
- candidate noun phrases: "Mechanistic Interpretation Closely", "NTERPRETATION Figure"
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "3.3 Mechanistic Interpretation" or "4.2 Deductiv… → "3.3 Mechanistic Interpretation"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
3. I
NTERPRETATION
Figure 3.1:
Syntax tree recovered from BERT representations with a structural probe.
Example by Hewitt and Manning (2019).
analogy tasks due to their ambiguity. And while analogical reasoning is fundamental
to humans, analogies cannot be represented as binary inference rules. Rogers et al.
(2017) also show experimentally that the word analogies only work when the source
and target vectors are close to each other. Other studies (Baroni et al. 2014; Levy and
Goldberg 2014) show that the relational similarities are not exclusive to neural word
embeddings but can also occur when using count-based word embedding strategies.
Structural probes looking for sentence-level features in contextual word representa-
tions are arguably less famous than relational probes on word embeddings but there
are some influential works. Hewitt and Manning (2019) develop a structural probe
that tests if entire syntax trees can be extracted as a (learned) linear transformation
from ELMo and BERT word representations. They assume that the number of edges
between words (the depth of the parse tree) may be encoded in the representation as
an L2 norm, finding out that it is indeed to some extent a structural property of the
word representation space. An example for such a tree from their work is presented in
Figure 3.1.
Ethayarajh (2019) investigates embeddings of BERT, ELMo, and GPT-2 with respect
to how contextual they are. The author tests (among some other things) how similar
the representations of the same word are in different contexts, and finds that in the
upper layers, the cosine similarity decreases, suggesting that the upper layers are more
task-specific. Kornblith et al. (2019) measure representational similarity between NNs
trained from scratch on image classification datasets with a measure that is invariant
to invertible linear transformations. They find that the representations of different
datasets are similar in early, but not in higher layers.
It is an intuitive idea that the positioning of representations in the vector space, shaped
by statistical patterns from the training corpus, is meaningful. However, structural
probes have the practical limitation that they require strong assumptions on how the
properties of interest are encoded in the representation. Those assumptions are only
possible to make for certain properties that can be translated into a geometric relation.
3.3
Mechanistic Interpretation
Closely related to the structural probes, mechanistic interpretability aims at a fine-
grained understanding of models at the neuron level. This fie
…
```
</details>

## Page 42 — ""

- address: `ohf://fold-bench/pages/000042`  ·  page cid: `c972cdb9d8da1f7c09682952bee343ec0168ae6e95ecf595536f6d1206077a3e`
- rendered page: `end-to-end/scaffold-images/p42.png`
- numbers in the prose (skip code):
    - `2017` — (2017) argue that downstream task evaluation is very coarse-grained and does not tell much about the types of information contained in the embeddings, and there…
    - `2018` — Bowman (2018) probe representations learned with different learning objectives, including language modeling and translation, for syntactic tasks, finding that t…
    - `2019` — (2019) use probing classifiers for tasks that require different sorts of syntactic information, from linear information like a word’s position to hierarchical…
    - `2022` — (2022) analyse the pre-training dynamics of a multilingual model by probing various training checkpoints.
    - `2023` — (2023) probe if the embedding space of a model encodes the answer- ability of a question with the help of a dataset that contains both answerable and unanswerab…
    - `3` — 3.4.3 Methodology Several works have raised concerns that a powerful probe may simply learn the task by storing information in its own parameters, rather than e…
    - `3.4` — 3.4.3 Methodology Several works have raised concerns that a powerful probe may simply learn the task by storing information in its own parameters, rather than e…
- candidate noun phrases: "Methodology Several"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
3. I
NTERPRETATION
from POS tags over syntactic dependencies, named entities and semantic roles to
coreference information. Y. Lin et al. (2019) use probing classifiers for tasks that
require different sorts of syntactic information, from linear information like a word’s
position to hierarchical information like the main auxiliary or the subject noun of
a sentence. Their results indicate that embeddings from lower layers contain more
linear information while higher layers contain more complex hierarchical features.
That semantics can be found in the higher layer while the lower layers contain more
syntactic information is also reported by Raganato and Tiedemann (2018) who use
probing classifiers on part-of-speech tagging, chunking, named entity recognition
and semantic tagging on Transformer encoders for machine translation. Similarly,
Jawahar et al. (2019) demonstrate that BERT learns surface features at the bottom
layers, syntactic features in the middle layers, and semantic features in the top layers,
suggesting that BERT requires deeper layers to learn long-distance features.
Blevins et al. (2022) analyse the pre-training dynamics of a multilingual model by
probing various training checkpoints. They find that while monolingual capabilities
are acquired early, cross-lingual capabilities emerge later in training, to a varying
degree depending on the language pair. They also find that linguistic knowledge
wanders from higher to lower layers during pre-training. K. Zhang and S. Bowman
(2018) probe representations learned with different learning objectives, including
language modeling and translation, for syntactic tasks, finding that the representations
trained on bidirectional language modelling contain the most useful information.
Probing has also been used for the evaluation of document or sentence embeddings.
Adi et al. (2017) argue that downstream task evaluation is very coarse-grained and
does not tell much about the types of information contained in the embeddings, and
therefore, generalisable conclusions cannot be drawn. They therefore introduce a set
of three tasks that capture the most basic properties of a sentence: word order, word
content, and length. Conneau et al. (2018) probe sentence embeddings for ten simple
linguistic features, finding that encoders contain a wide range of linguistic information
compared to baselines. They argue that for these simple tasks, it is easier to control for
biases than for downstream tasks.
Slobodkin et al. (2023) probe if the embedding space of a model encodes the answer-
ability of a question with the he
…
```
</details>

## Page 50 — "4.2 Deductive Procedure"

- address: `ohf://fold-bench/pages/000050`  ·  page cid: `b768e7dd8ba9324700e50e3ccd6462d7b598fca16a7d84c0d122c61d346e7d81`
- rendered page: `end-to-end/scaffold-images/p50.png`
- numbers in the prose (skip code):
    - `1986` — Long before deep learning became popular, procedural explanations have been used in artificial intelligence to learn generalization by capturing the structural …
    - `1988` — Long before deep learning became popular, procedural explanations have been used in artificial intelligence to learn generalization by capturing the structural …
    - `2018` — (2018) employ explanations in the form of decision sets , mappings of inputs to outputs via a set of rules.
    - `2022` — 2022) tree for science question answering.
    - `4.2` — E XPLANATIONS Figure 4.2: Example for a deductive explanation: A constrained METGEN (Hong et al.
- candidate noun phrases: "Deductive Procedure Deductive", "XPLANATIONS Figure"
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "6.2.1 Understanding the Internal Processes of LL… → "4.2 Deductive Procedure"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
4. E
XPLANATIONS
Figure 4.2:
Example for a deductive explanation: A constrained METGEN (Hong et al.
2022) tree for science question answering. The question in this example was:
How
might eruptions affect plants?
, the answer, as shown in green in the figure:
Eruptions
can cause plants to die
. Orange denotes facts; blue intermediate conclusions. Figure
adapted from Hong et al. (2022).
For the comparison with human rationales, there exist datasets with annotations for
relevant input spans. The benchmark ERASER (DeYoung et al. 2020) collects seven
such datasets. The authors of ERASER propose to measure agreement with these
human annotations, but also faithfulness scores based in sufficiency and comprehen-
siveness as the goal of outputting attribution scores is to create not only
plausible
but
also
faithful
explanations. The importance of the disentanglement between the two has
been noted by Jacovi and Goldberg (2020) who point out that human ratings or gold
standards are inappropriate for faithfulness evaluation as plausibility to humans does
not indicate what a machine learning model is doing internally.
4.2
Deductive Procedure
Deductive procedures are grounded in the input and provide step-by-step rules that
lead to the prediction. They are less common because they are only applicable to a
small share of NLP tasks (Tan 2022) but provide complete inference chains where
all intermediate steps can be checked. Long before deep learning became popular,
procedural explanations have been used in artificial intelligence to learn generalization
by capturing the structural relationship of a problem (DeJong and Mooney 1986; Lewis
1988; Mitchell et al. 1986).
Narayanan et al. (2018) employ explanations in the form of
decision sets
, mappings of
inputs to outputs via a set of rules. In user studies, they search for explanations that
humans can utilize best, finding that more complex explanations are harder to process
and less satisfactory. Hong et al. (2022) build constrained trees consisting of entailment
steps for science question answering. An example for such a tree generated with their
METGEN system can be found in Figure 4.2. Ling et al. (2017) and Jie et al. (2022)
generate the intermediate steps necessary to solve math word problems. This is similar
to the recently famous Chain-of-Thought (CoT) generation (Wei et al. 2022), where the
model generates intermediate reasoning steps prior to the prediction in a zero-shot
setting. However, in CoT, the completeness and correctness of the intermediate steps
is neither enforced nor typically evaluated; its main goal
…
```
</details>

## Page 51 — "4.3 Natural Language Explanations"

- address: `ohf://fold-bench/pages/000051`  ·  page cid: `ce743d71a7b134094feeeecc4fb78e3ce6e56cec9f2f03558e187803bb2ea699`
- rendered page: `end-to-end/scaffold-images/p51.png`
- numbers in the prose (skip code):
    - `1` — Datasets with human-written free-text explanations 1 for the correct label were created for tasks like natural language inference (Camburu et al.
    - `1963` — To understand how rare purely formal reasoning is in humans, we should consider that even discovery-focused parts of mathematics have informal components: They …
    - `2` — 2020b), it was only with the emergence of GPT-2 and other generative Transformer models that they gained more traction.
    - `2000` — One is applicability: The prediction problem needs to be fully formalisable, which is a very strong assumption as most dynamical systems are not characterisable…
    - `2006` — Humans, on the other hand, explain at different levels of abstraction, even if their understanding is coarse and fragmentary (Keil 2006).
    - `2014` — 2014) or approaches based on extractive summarisation (Atanasova et al.
    - `2018` — 2018) and multiple-choice question answering (Aggarwal et al.
    - `2019` — 2019), along with models fine-tuned to imitate these explanations.
    - `2020` — 2020b), it was only with the emergence of GPT-2 and other generative Transformer models that they gained more traction.
    - `4.3` — 4.3 Natural Language Explanations Generating natural language explanations has gained relatively little attention until a few years ago.
- candidate noun phrases: "Natural Language Explanations Generating", "Natural Language Explanations There", "Yongfeng Zhang"
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "4.3 Natural Language Explanations" or "4.2 Deduc… → "4.3 Natural Language Explanations"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
4.3. Natural Language Explanations
There are two main limiting factors in deductive procedures. One is applicability: The
prediction problem needs to be fully formalisable, which is a very strong assumption
as most dynamical systems are not characterisable as (interpretable) computations
(Cummins 2000). Also, procedures are limited to problems solvable by simple and
transparent algorithms, such as decision trees. Humans, on the other hand, explain at
different levels of abstraction, even if their understanding is coarse and fragmentary
(Keil 2006). To understand how rare purely formal reasoning is in humans, we should
consider that even discovery-focused parts of mathematics have informal components:
They require the refinement of guesses by speculation and criticism, heuristics, and
exploration (Lakatos 1963).
The second limitation of deductive procedures is their understandability: If the ex-
planation exceeds a certain length, it will be hard for a human user to follow. The
causal and relational complexity of the real world would however require deductive
procedures of unbound length, making understandable procedures utopian.
4.3
Natural Language Explanations
Generating natural language explanations has gained relatively little attention until a
few years ago. While some works use more restrictive techniques to generate textual
explanations, such as template-based approaches (N. Wang et al. 2018; Yongfeng Zhang
et al. 2014) or approaches based on extractive summarisation (Atanasova et al. 2020b),
it was only with the emergence of GPT-2 and other generative Transformer models
that they gained more traction. Datasets with human-written free-text explanations
1
for the correct label were created for tasks like natural language inference (Camburu
et al. 2018) and multiple-choice question answering (Aggarwal et al. 2021; Rajani et al.
2019), along with models fine-tuned to imitate these explanations.
Natural language explanations address many of the limitations that attribution meth-
ods and procedural methods have. They are easily accessible to human users than
other forms of explanation, especially to end users without a technical background.
Moreover, they can incorporate forms of reasoning not covered by the other methods:
They allow for the inclusion of any input-external knowledge and any type of rea-
soning that can be expressed in natural language, while not being restricted to tasks
where it is possible to provide a complete reasoning path.
The incorporation of free-text explanations in the learning process has in some cases
lead to an increase in
…
```
</details>

## Page 55 — ""

- address: `ohf://fold-bench/pages/000055`  ·  page cid: `56e780a5880f2c22aa486ccfa1d64fd0d8b960cfe2787b3fb0686bd324bdddd7`
- rendered page: `end-to-end/scaffold-images/p55.png`
- numbers in the prose (skip code):
    - `1` — Natural Language Explanations Training Phase 1 Training Phase 2 Figure 4.3: Graphical representation of the categorisation proposed by Hase et al.
    - `2` — Natural Language Explanations Training Phase 1 Training Phase 2 Figure 4.3: Graphical representation of the categorisation proposed by Hase et al.
    - `2019` — 2019) or prior to the label prediction model (Camburu et al.
    - `2020` — As Latcinnik and Berant (2020) acknowledge, this is a weak form of explanation as the prediction process cannot influence the explanation generation.
    - `2021` — (2021) show that self-rationalising models have a higher performance than serial-task architectures.
    - `2022` — (2022) were the first to propose a few-shot approach that jointly generates model predictions and explanations.
    - `3` — After the release of GPT-3 and the increasing capabilities of models in in-context learning, Marasovic et al.
    - `4.3` — Natural Language Explanations Training Phase 1 Training Phase 2 Figure 4.3: Graphical representation of the categorisation proposed by Hase et al.
- candidate noun phrases: "Natural Language Explanations Training", "Training Phase"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
4.3. Natural Language Explanations
Training Phase 1
Training Phase 2
Figure 4.3:
Graphical representation of the categorisation proposed by Hase et al.
(2020).
x
is the input,
y
the output and
e
the explanation. Figure by Hase et al. (2020).
4.3.2
Approaches
In the earlier works on generating free-text explanations, the explanation is generated
either completely independently of (Rajani et al. 2019) or prior to the label prediction
model (Camburu et al. 2018; Latcinnik and Berant 2020). As Latcinnik and Berant
(2020) acknowledge, this is a weak form of explanation as the prediction process cannot
influence the explanation generation. Hase et al. (2020) introduce a categorisation of
the approaches, calling the former type of approach that conditions the explanation
only on inputs
reasoning
(RE) mode, and the latter that considers the input as well as
the label
rationalising
(RA) mode. They also consider if the explanations are used as
an additional input in a pipeline model which they call a
serial-task
(ST) approach or
explanations are generated jointly with the labels, called a
multi-task
(MT) approach.
A graphical representation of the four approaches is presented in Figure 4.3. Hase
et al. (2020)’s multi-task reasoning mode has subsequently become known as
self-
rationalising models
, the term that I adopt in this thesis.
With the increasing multi-task capabilities of LLMs, self-rationalising models have
become increasingly common. Narang et al. (2020) show that they can successfully
generate labels and explanations at the same time, and that the self-rationalisation
capabilities can, to some extent, even be transferred to other datasets. Wiegreffe et al.
(2021) show that self-rationalising models have a higher performance than serial-task
architectures.
These first approaches all used fine-tuned models. After the release of GPT-3 and the
increasing capabilities of models in in-context learning, Marasovic et al. (2022) were
the first to propose a few-shot approach that jointly generates model predictions and
explanations. They explore how the prompt should be formatted for such a setup, and
show that while the prompt has a significant impact on the performance, humans rate
their generated model explanations as significantly less plausible than human-written
explanations.
These findings have also been exploited for the generation of datasets. Synthetic, LLM-
generated datasets are becoming more common as they reach a reasonable quality for
many NLP tasks and applications with a fraction of the cost, and the same appears to be
true for LLM-generated n
…
```
</details>

## Page 66 — "6.2.1 Understanding the Internal Processes of LLMs"

- address: `ohf://fold-bench/pages/000066`  ·  page cid: `873be58ef1bb5c1a1f5af6a62e7a767e63607b9cf047fd089e3bbd0cebaaed61`
- rendered page: `end-to-end/scaffold-images/p66.png`
- numbers in the prose (skip code):
    - `1` — 6.2.1 Understanding the Internal Processes of LLMs Probing classifiers have given us valuable insights into how linguistic information is structured within an L…
    - `3,` — However, like with many other interpretability methods, as outlined in Chapter 3, those insights are coarse and need to be interpreted relative to a baseline or…
    - `6.2` — 6.2 Outlook Finally, I outline some questions that I consider central for future work in interpretable and explainable NLP, based on our research on these topic…
- candidate noun phrases: "Internal Processes", "LLMs Probing", "Outlook Finally"
- auto-generated for this page:
    - [locate] This page has a section heading for exactly one of these refactorings: "3.2 Structural Probes" or "6.2.1 Understanding t… → "6.2.1 Understanding the Internal Processes of LLMs"

<details><summary>full page text (verbatim — the authoring evidence)</summary>

```
6. C
ONCLUSION
where syntactic information is located in the model from the middle layers, where the
overall performance is the highest, to the earliest layers, which contribute the most
new information. Both the focus on extrapolation capabilities and the focus on local
information gains provide new perspectives to the probing method which can help us
to gain more robust insights.
In the field of explainability, we focused on the properties and utility of free-text
explanations generated by LLMs. We showed that human ratings, particularly of the
factual correctness of the explanations, are not indicative of the performance when
using the generated explanations in a downstream model. For the latter, including
more novel information in the explanations appears to be beneficial, but this comes at
the risk of including incorrect or irrelevant information, which human raters punish
decisively. We further examined which common properties of human explanations
are commonly reflected in LLM-generated explanations. The results of our annotation
study indicate that they often list incomplete sets of contributing reasons as well as
illustrative examples.
Interpretation and explanation methods can complement each other in shaping a better
understanding of LLMs. The former methods give us a high-level understanding of
how the individual tokens are contextualised and, layer for layer, form a representation
useful for many applications. The latter methods give us an idea of the context and
reasoning accessible to the model when making a prediction, even if the explanations
are not faithful to the model’s decision process. Together with an understanding
of the LLMs’ architecture and training objectives, such methods make it possible
to achieve a coarse understanding of the decision-making process and be able to
predict the models’ behaviour to a certain extent. This understanding is insufficient
to allow for the deployment of LLMs for high-stakes decisions without a human in
the loop. However, it has the potential to enable developers and users to make better
decisions on whether the model will be able to perform a certain task, whether its
decision-making is sufficiently robust, and how it can be improved.
6.2
Outlook
Finally, I outline some questions that I consider central for future work in interpretable
and explainable NLP, based on our research on these topics.
6.2.1
Understanding the Internal Processes of LLMs
Probing classifiers have given us valuable insights into how linguistic information is
structured within an LLM and how this representation is forme
…
```
</details>
